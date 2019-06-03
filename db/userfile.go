package db

import (
	"fmt"
	mydb "github.com/lsj575/filestore-server/db/mysql"
	"time"
)
// 用户文件表结构体
type UserFile struct {
	Username string
	FileHash string
	FileName string
	FileSize int64
	UploadAt string
	LastUpdated string
}

// 更新用户文件表
func OnUserFileUploadFinished(username string, filehash string, filename string, filesize int64) bool {
	stmt, err := mydb.DBConn().Prepare(
		"INSERT IGNORE INTO tbl_user_file (`user_name`, `file_sha1`, `file_name`, `file_size`, `upload_at`) " +
			"values (?, ?, ?, ?, ?)")
	if err != nil {
		return false
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, filehash, filename, filesize, time.Now())
	if err != nil {
		return false
	}
	return true
}

func QueryUserFileMetas(username string, limit int) ([]UserFile, error) {
	stmt, err := mydb.DBConn().Prepare(
		"SELECT file_sha1, file_name, file_size, upload_at, last_upload FROM tbl_user_file WHERE user_name = ? LIMIT ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(username, limit)
	if err != nil {
		return nil, err
	}

	var userFiles []UserFile
	for rows.Next() {
		uFile := UserFile{}
		err = rows.Scan(&uFile.FileHash, &uFile.FileName, &uFile.FileSize, &uFile.UploadAt, &uFile.LastUpdated)
		if err != nil {
			fmt.Println(err.Error())
			break
		}
		userFiles = append(userFiles, uFile)
	}
	return userFiles, nil
}