package util

import (
	"fmt"
	"os"
)

type FileInfoResult int

const (
	FileInfoResultError FileInfoResult = iota
	FileInfoResultNotExists
	FileInfoResultExistFile
	FileInfoResultExistDir
)

func GetFileInfo(path string) (FileInfoResult, error) {
	if fileInfo, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return FileInfoResultError, err
		}
		return FileInfoResultNotExists, nil
	} else if fileInfo.IsDir() {
		return FileInfoResultExistDir, nil
	} else {
		return FileInfoResultExistFile, nil
	}
}

type MkdirIfNotExistsResult int

const (
	MkdirIfNotExistsResultError MkdirIfNotExistsResult = iota
	MkdirIfNotExistsResultExistsFile
	MkdirIfNotExistsResultExists
	MkdirIfNotExistsResultOK
)

func MkdirIfNotExists(path string) (MkdirIfNotExistsResult, error) {
	if fileInfo, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return MkdirIfNotExistsResultError, err
		}
		return MkdirIfNotExistsResultOK, nil
	} else {
		if !fileInfo.IsDir() {
			return MkdirIfNotExistsResultExistsFile, nil
		}
		return MkdirIfNotExistsResultExists, nil
	}
}

func GetFileSize(path string) (int64, error) {
	if fileInfo, err := os.Stat(path); os.IsNotExist(err) {
		return 0, fmt.Errorf("文件不存在：%s", err)
	} else if fileInfo.IsDir() {
		return 0, fmt.Errorf("此路径不是文件：%s", err)
	} else {
		return fileInfo.Size(), nil
	}
}
