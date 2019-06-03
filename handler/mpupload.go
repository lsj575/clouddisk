package handler

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	rPool "github.com/lsj575/filestore-server/cache/redis"
	dblayer "github.com/lsj575/filestore-server/db"
	"github.com/lsj575/filestore-server/util"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// 初始化信息
type MultiPartUploadInfo struct {
	FileHash string
	FileSize int64
	UploadID string
	ChunkSize int
	ChunkCount int
}

// 初始化分块上传
func InitialMultiPartUploadHandler(w http.ResponseWriter, r *http.Request)  {
	// 解析用户请求信息
	r.ParseForm()
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize, err := strconv.Atoi(r.Form.Get("filesize"))
	if err != nil {
		w.Write(util.NewRespMsg(-1, "Invalid params", nil).JSONBytes())
		return
	}

	// 获得redis连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 生成分块上传的初始化信息
	upInfo := MultiPartUploadInfo{
		FileHash:   filehash,
		FileSize:   int64(filesize),
		UploadID:   username + fmt.Sprintf("%x", time.Now().UnixNano()),
		ChunkSize:  5 * 1024 * 1024, // 5MB
		ChunkCount: int(math.Ceil(float64(filesize) / (5 * 1024 * 1024))),
	}
	// 将初始化信息写入redis缓存
	rConn.Do("HSET", "MP_" + upInfo.UploadID, "chunkcount", upInfo.ChunkCount)
	rConn.Do("HSET", "MP_" + upInfo.UploadID, "filehash", upInfo.FileHash)
	rConn.Do("HSET", "MP_" + upInfo.UploadID, "filesize", upInfo.FileSize)
	// 将初始化信息返回给客户端
	w.Write(util.NewRespMsg(0, "OK", upInfo).JSONBytes())
}

// 分块上传
func UploadPartHandler(w http.ResponseWriter, r *http.Request) {
	// 解析用户请求参数
	r.ParseForm()
	username := r.Form.Get("username")
	uploadID := r.Form.Get("uploadid")
	chunkIndex := r.Form.Get("index")

	// 获得redis连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 根据当前用户，获取文件句柄，用于存储分块内容
	fPath := "./data/" + uploadID + "/" + chunkIndex
	os.MkdirAll(path.Dir(fPath), 0744)
	file, err := os.Create(fPath)
	if err != nil {
		w.Write(util.NewRespMsg(-1, "Upload part failed", nil).JSONBytes())
		return
	}
	defer file.Close()
	buf := make([]byte, 1024 * 1024)
	for {
		n, err := r.Body.Read(buf)
		file.Write(buf[:n])
		if err != nil {
			break
		}
	}
	// 更新redis缓存
	rConn.Do("HSET", "MP_" + uploadID, "chkidx_" + chunkIndex, 1)
	// 返回结果给客户端
	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}

// 通知上传合并
func CompleteUploadHandler(w http.ResponseWriter, r *http.Request)  {
	// 解析用户参数
	r.ParseForm()
	uploadID := r.Form.Get("uploadid")
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize, _ := strconv.Atoi(r.Form.Get("filesize"))
	filename := r.Form.Get("filename")
	// 获得redis连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// uploadid查询redis并判断是否所有分块都上传完成
	data, err := redis.Values(rConn.Do("HGETALL", "MP_"+uploadID))
	if err != nil {
		w.Write(util.NewRespMsg(-1, "Complete upload failed", nil).JSONBytes())
		return
	}
	totalCount := 0
	chunkCount := 0
	for i := 0; i < len(data); i += 2 {
		k := string(data[i].([]byte))
		v := string(data[i+1].([]byte))
		if k == "chunkcount" {
			totalCount, _ = strconv.Atoi(v)
		} else if strings.HasPrefix(k, "chkidx_") && v == "1" {
			chunkCount ++
		}
	}
	if totalCount != chunkCount {
		w.Write(util.NewRespMsg(-1, "invalid request", nil).JSONBytes())
		return
	}
	// 合并分块
	file, err := os.Create("./static/file/" + filename)
	if err != nil {
		w.Write(util.NewRespMsg(-1, "Failed to complete upload", nil).JSONBytes())
		return
	}
	defer file.Close()
	for i := 1; i <= totalCount; i++ {
		f, err := os.Open("./data/" + uploadID + "/" + strconv.Itoa(i))
		if err != nil {
			w.Write(util.NewRespMsg(-1, "Failed to complete upload", nil).JSONBytes())
			return
		}
		defer f.Close()
		buf := make([]byte, 1024 * 1024)
		for {
			n, err := f.Read(buf)
			file.Write(buf[:n])
			if err != nil {
				break
			}
		}
	}
	// 更新唯一文件表和用户文件表
	dblayer.OnFileUploadFinished(filehash, filename, int64(filesize), "")
	dblayer.OnUserFileUploadFinished(username, filehash, filename, int64(filesize))
	// 响应处理结果
	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}
