package db

import (
	"database/sql"
	"fmt"
	mydb "github.com/lsj575/filestore-server/db/mysql"
)

type TableFile struct {
	FileHash string
	FileName sql.NullString
	FileSize sql.NullInt64
	FileAddr sql.NullString
}

// 文件上传完成，保存meta
func OnFileUploadFinished(filehash string, filename string, filesize int64, fileaddr string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"INSERT ignore INTO tbl_file (`file_sha1`, `file_name`, `file_size`, `file_addr`, `status`)" +
			"values (?, ?, ?, ?, 1)")
	if err != nil {
		fmt.Println("Failed to prepare statement, err: ", err)
		return false
	}
	defer stmt.Close()

	result, err := stmt.Exec(filehash, filename, filesize, fileaddr)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	if ra, err := result.RowsAffected(); err == nil {
		if ra <= 0 {
			fmt.Printf("File with hash %s has been uploaded before\n", filehash)
			return false
		}
		return true
	}

	return false
}

// 从MySQL获取文件元信息
func GetFileMeta(filehash string) (*TableFile, error) {
	stmt, err := mydb.DBConn().Prepare(
		"SELECT file_sha1, file_name, file_size, file_addr FROM tbl_file " +
			"WHERE file_sha1 = ? AND status = 1 LIMIT 1")
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer stmt.Close()

	tFile := TableFile{}
	err = stmt.QueryRow(filehash).Scan(&tFile.FileHash, &tFile.FileName, &tFile.FileSize, &tFile.FileAddr)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return &tFile, nil
}
