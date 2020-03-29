package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gabstv/go-bsdiff/pkg/bspatch"

	"github.com/ganlvtech/go-dir-bsdiff/patch"
	"github.com/ganlvtech/go-dir-bsdiff/util"
)

const (
	HelpTemplate = "使用方法：\n\n" +
		"    patch.exe 旧文件夹路径 新文件夹路径 差异文件夹路径\n\n" +
		"如果新文件夹路径不存在则会自动创建文件夹\n\n" +
		"使用到的开源软件：\n\n" +
		"    Pure Go bsdiff and bspatch libraries and CLI tools.\n" +
		"        https://github.com/gabstv/go-bsdiff\n\n" +
		"本程序使用 Go 语言开发，由 %s 生成"
	CopyBufferSize = 1024 * 1024
)

var Help = fmt.Sprintf(HelpTemplate, runtime.Version())

func Patch(newDirAbsPath, oldDirAbsPath, diffDirAbsPath string, fileName string) {
	oldFilePath := oldDirAbsPath + "\\" + fileName
	newFilePath := newDirAbsPath + "\\" + fileName
	newFileDirPath := filepath.Dir(newFilePath)
	if mkdirResult, err := util.MkdirIfNotExists(newFileDirPath); err != nil {
		log.Fatal(newFileDirPath, "创建文件夹失败：", err)
	} else if mkdirResult == util.MkdirIfNotExistsResultExistsFile {
		log.Fatal(newFileDirPath, "路径存在，但不是文件夹")
	} else if mkdirResult == util.MkdirIfNotExistsResultOK {
		log.Println("创建文件夹成功")
	}
	diffFilePath := diffDirAbsPath + "\\" + fileName + patch.BsDiffFileSuffix
	err := bspatch.File(oldFilePath, newFilePath, diffFilePath)
	if err != nil {
		log.Fatal(fileName, "更新文件失败：", err)
	}
	log.Println(fileName, "更新文件成功")
}

func PartCopyOld(newFileWriter io.Writer, oldFileReader io.Reader, bulkSize int) error {
	buf := make([]byte, bulkSize)
	_, err := oldFileReader.Read(buf)
	if err != nil {
		return err
	}
	_, err = newFileWriter.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func PartCopyNew(newFileWriter io.Writer, diffNewFilePath string) error {
	diffNewFileReader, err := os.Open(diffNewFilePath)
	if err != nil {
		return err
	}
	defer diffNewFileReader.Close()
	buf := make([]byte, CopyBufferSize)
	_, err = io.CopyBuffer(newFileWriter, diffNewFileReader, buf)
	if err != nil {
		return err
	}
	return nil
}

func PartPatch(newFileWriter io.Writer, oldFileReader io.Reader, diffFilePath string, bulkSize int) error {
	oldFileBytes := make([]byte, bulkSize)
	_, err := oldFileReader.Read(oldFileBytes)
	if err != nil {
		return err
	}
	diffBytes, err := ioutil.ReadFile(diffFilePath)
	if err != nil {
		return err
	}
	newBytes, err := bspatch.Bytes(oldFileBytes, diffBytes)
	if err != nil {
		return err
	}
	_, err = newFileWriter.Write(newBytes)
	if err != nil {
		return err
	}
	return nil
}

func AutoPartPatch(newFilePath, oldFilePath, partFileBasePath string, partOperations []string, bulkSize int) {
	newFileWriter, err := os.OpenFile(newFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(newFilePath, "打开新文件失败")
	}
	defer newFileWriter.Close()
	oldFileReader, err := os.Open(oldFilePath)
	if err != nil {
		log.Fatal(oldFilePath, "打开旧文件失败")
	}
	defer newFileWriter.Close()

	for i, partOperation := range partOperations {
		partIndex := i + 1
		switch partOperation {
		case patch.OperationTypeCopyOld:
			err := PartCopyOld(newFileWriter, oldFileReader, bulkSize)
			if err != nil {
				log.Fatal(newFilePath, "复制旧文件失败")
			}
		case patch.OperationTypeCopyNew:
			_, err = oldFileReader.Seek(int64(bulkSize), 1)
			if err != nil {
				log.Fatal(newFilePath, "复制旧文件块失败")
			}
			partNewFilePath := patch.GetPartNewFileName(partFileBasePath, partIndex)
			err = PartCopyNew(newFileWriter, partNewFilePath)
			if err != nil {
				log.Fatal(newFilePath, "复制新文件失败")
			}
		case patch.OperationTypePatch:
			partDiffFilePath := patch.GetPartDiffFileName(partFileBasePath, partIndex)
			err := PartPatch(newFileWriter, oldFileReader, partDiffFilePath, bulkSize)
			if err != nil {
				log.Fatal(newFilePath, "更新文件失败")
			}
		default:
			log.Fatal("未知操作", partOperation)
		}
	}
}

func getArgs() (oldDirAbsPath, newDirAbsPath, diffDirAbsPath string, err error) {
	if len(os.Args) <= 3 {
		return "", "", "", fmt.Errorf("调用参数数量不足\n\n%s", Help)
	}
	oldDir := os.Args[1]
	newDir := os.Args[2]
	diffDir := os.Args[3]
	if oldFileInfo, err := util.GetFileInfo(oldDir); err != nil {
		return "", "", "", err
	} else if oldFileInfo == util.FileInfoResultNotExists {
		return "", "", "", fmt.Errorf("旧版路径 %s 不存在", oldDir)
	} else if oldFileInfo == util.FileInfoResultExistFile {
		return "", "", "", fmt.Errorf("旧版路径 %s 不是文件夹", oldDir)
	}
	if diffFileInfo, err := util.GetFileInfo(diffDir); err != nil {
		return "", "", "", err
	} else if diffFileInfo == util.FileInfoResultNotExists {
		return "", "", "", fmt.Errorf("新版路径 %s 不存在", diffDir)
	} else if diffFileInfo == util.FileInfoResultExistFile {
		return "", "", "", fmt.Errorf("新版路径 %s 不是文件夹", diffDir)
	}
	if diffFileInfo, err := util.GetFileInfo(diffDir); err != nil {
		return "", "", "", err
	} else if diffFileInfo == util.FileInfoResultExistFile {
		return "", "", "", fmt.Errorf("输出差异路径 %s 不是文件夹", diffDir)
	}
	oldDirAbsPath, err = filepath.Abs(oldDir)
	if err != nil {
		return "", "", "", fmt.Errorf("获取旧版路径 %s 的绝对路径失败：%s", oldDir, err)
	}
	newDirAbsPath, err = filepath.Abs(newDir)
	if err != nil {
		return oldDirAbsPath, "", "", fmt.Errorf("获取新版路径 %s 的绝对路径失败：%s", newDir, err)
	}
	diffDirAbsPath, err = filepath.Abs(diffDir)
	if err != nil {
		return oldDirAbsPath, newDirAbsPath, "", fmt.Errorf("获取输出差异路径 %s 的绝对路径失败：%s", diffDir, err)
	}
	return oldDirAbsPath, newDirAbsPath, diffDirAbsPath, nil
}

func main() {
	oldDirAbsPath, newDirAbsPath, diffDirAbsPath, err := getArgs()
	if err != nil {
		log.Fatal("读取补丁描述文件错误：", err)
	}
	patchJSONPath := filepath.Join(diffDirAbsPath, patch.ManifestFileName)
	file, err := ioutil.ReadFile(patchJSONPath)
	if err != nil {
		log.Fatal("读取补丁描述文件错误：", err)
	}
	patchManifest := &patch.Manifest{}
	err = json.Unmarshal(file, patchManifest)
	if err != nil {
		log.Fatal("读取补丁描述文件错误：", err)
	}
	for fileName, fileMD5 := range patchManifest.OldMd5 {
		oldFilePath := oldDirAbsPath + "\\" + fileName
		oldFileMD5, err := util.FileMD5(oldFilePath)
		if err != nil {
			log.Fatal(oldFilePath, "文件 md5 计算错误：", err)
		}
		if oldFileMD5 != fileMD5 {
			log.Fatal(oldFilePath, "旧版文件 md5 不正确，无法进行差异更新")
		}
	}
	for fileName, operation := range patchManifest.Patches {
		newFilePath := newDirAbsPath + "\\" + fileName
		newFileDirPath := filepath.Dir(newFilePath)
		if mkdirResult, err := util.MkdirIfNotExists(newFileDirPath); err != nil {
			log.Fatal(newFileDirPath, "创建文件夹失败：", err)
		} else if mkdirResult == util.MkdirIfNotExistsResultExistsFile {
			log.Fatal(newFileDirPath, "路径存在，但不是文件夹")
		} else if mkdirResult == util.MkdirIfNotExistsResultOK {
			log.Println("文件夹创建成功")
		}

		if operation == patch.OperationTypeCopyOld {
			oldFilePath := oldDirAbsPath + "\\" + fileName
			err := util.CopyFile(newFilePath, oldFilePath)
			if err != nil {
				log.Fatal(fileName, "复制文件错误", err)
			}
			log.Println(fileName, "复制成功")
		} else if operation == patch.OperationTypeCopyNew {
			diffNewFilePath := diffDirAbsPath + "\\" + fileName
			err := util.CopyFile(newFilePath, diffNewFilePath)
			if err != nil {
				log.Fatal(fileName, "复制文件错误", err)
			}
			log.Println(fileName, "复制成功")
		} else if operation == "" {
			log.Fatal("操作为空")
		} else {
			partOperations := strings.Split(operation, ",")
			if len(partOperations) > 1 {
				oldFilePath := oldDirAbsPath + "\\" + fileName
				partFileBasePath := diffDirAbsPath + "\\" + fileName
				AutoPartPatch(newFilePath, oldFilePath, partFileBasePath, partOperations, patchManifest.BulkSize)
			} else {
				partOperation := partOperations[0]
				if partOperation != patch.OperationTypePatch {
					log.Fatal("未知操作", partOperation)
				}
				Patch(newDirAbsPath, oldDirAbsPath, diffDirAbsPath, fileName)
			}
		}
	}
	for fileName, fileMD5 := range patchManifest.NewMd5 {
		newFilePath := newDirAbsPath + "\\" + fileName
		newFileMD5, err := util.FileMD5(newFilePath)
		if err != nil {
			log.Fatal(newFilePath, "文件 md5 计算错误：", err)
		}
		if newFileMD5 != fileMD5 {
			log.Fatal(newFilePath, "新版文件 md5 不正确，差异更新错误")
		}
	}
}
