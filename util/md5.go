package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func BytesMD5(data []byte) string {
	md5sum :=  md5.Sum(data)
	result := hex.EncodeToString(md5sum[:])
	return result
}

func FileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("计算文件 MD5 时打开文件错误：%s", err)
	}
	defer f.Close()

	md5hash := md5.New()
	if _, err := io.Copy(md5hash, f); err != nil {
		return "", fmt.Errorf("计算文件 MD5 时读取文件错误：%s", err)
	}

	md5sum := md5hash.Sum(nil)
	result := hex.EncodeToString(md5sum)
	return result, nil
}

func DirFilesMD5(dirAbsPath string) (map[string]string, error) {
	md5List := make(map[string]string)
	filePathPrefix := dirAbsPath + "\\"
	err := filepath.Walk(dirAbsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if !strings.HasPrefix(path, filePathPrefix) {
				return fmt.Errorf("路径 %s 不以 %s 为前缀", path, filePathPrefix)
			}
			relPath := path[len(filePathPrefix):]
			md5sum, err := FileMD5(path)
			if err != nil {
				return err
			}
			md5List[relPath] = md5sum
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历目录全部文件错误：%s", err)
	}
	return md5List, err
}
