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
	"sort"
	"strconv"
	"strings"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"

	"github.com/ganlvtech/go-dir-bsdiff/patch"
	"github.com/ganlvtech/go-dir-bsdiff/util"
)

const (
	HelpTemplate = "使用方法：\n\n" +
		"    diff.exe 旧文件夹路径 新文件夹路径 差异文件夹路径 [文件分块大小]\n\n" +
		"如果差异文件夹路径不存在则会自动创建文件夹\n\n" +
		"默认分块大小是 100MB，第四个参数默认为 104857600\n\n" +
		"使用到的开源软件：\n\n" +
		"    Pure Go bsdiff and bspatch libraries and CLI tools.\n" +
		"        https://github.com/gabstv/go-bsdiff\n\n" +
		"本程序使用 Go 语言开发，由 %s 生成"
	DefaultBulkSize = 100 * 1024 * 1024
)

var Help = fmt.Sprintf(HelpTemplate, runtime.Version())

func DoBsDiffPart(oldBytes []byte, newBytes []byte, diffFilePath string, diffNewFilePath string) (string, int, error) {
	oldBytesMD5 := util.BytesMD5(oldBytes)
	newBytesMD5 := util.BytesMD5(newBytes)
	if oldBytesMD5 == newBytesMD5 {
		return patch.OperationTypeCopyOld, 0, nil
	} else {
		newBytesSize := len(newBytes)
		diffBytes, err := bsdiff.Bytes(oldBytes, newBytes)
		if err != nil {
			return "", 0, fmt.Errorf("执行 bsdiff.File 错误：%s", err)
		}
		diffFileSize := len(diffBytes)
		if diffFileSize > newBytesSize {
			err = ioutil.WriteFile(diffNewFilePath, newBytes, 0644)
			if err != nil {
				return "", 0, fmt.Errorf("复制文件错误：%s", err)
			}
			return patch.OperationTypeCopyNew, newBytesSize, nil
		} else {
			err = ioutil.WriteFile(diffFilePath, diffBytes, 0644)
			if err != nil {
				return "", 0, fmt.Errorf("写入差异文件错误：%s", err)
			}
			return patch.OperationTypePatch, diffFileSize, nil
		}
	}
}

func DoBsDiff(oldFilePath, newFilePath, diffFileBasePath string, bulkSize int) (string, int, error) {
	oldFileSize, err := util.GetFileSize(oldFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("获取旧版文件大小错误：%s", err)
	}
	newFileSize, err := util.GetFileSize(newFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("获取新版文件大小错误：%s", err)
	}
	oldFileReader, err := os.Open(oldFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("打开旧版文件错误：%s", err)
	}
	defer oldFileReader.Close()
	newFileReader, err := os.Open(newFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("打开新版文件错误：%s", err)
	}
	defer newFileReader.Close()
	oldBytes := make([]byte, bulkSize)
	newBytes := make([]byte, bulkSize)
	if int(oldFileSize) <= bulkSize && int(newFileSize) <= bulkSize {
		oldBytesRead, err := oldFileReader.Read(oldBytes)
		if err != nil {
			if err != io.EOF {
				return "", 0, fmt.Errorf("读取旧版文件错误：%s", err)
			}
		}
		newBytesRead, err := newFileReader.Read(newBytes)
		if err != nil {
			if err != io.EOF {
				return "", 0, fmt.Errorf("读取新版文件错误：%s", err)
			}
		}
		diffFilePath := diffFileBasePath + patch.BsDiffFileSuffix
		diffNewFilePath := diffFileBasePath
		return DoBsDiffPart(oldBytes[:oldBytesRead], newBytes[:newBytesRead], diffFilePath, diffNewFilePath)
	} else {
		partIndex := 1
		oldFileFinished := false
		newFileFinished := false
		oldBytesReadSum := 0
		newBytesReadSum := 0
		resultParts := make([]string, 0)
		resultSize := 0
		for !oldFileFinished && !newFileFinished {
			oldBytesRead, err := oldFileReader.Read(oldBytes)
			if err != nil {
				if err != io.EOF {
					return "", 0, fmt.Errorf("读取旧版文件错误：%s", err)
				}
				oldFileFinished = true
				break
			}
			oldBytesReadSum += oldBytesRead
			if oldBytesReadSum >= int(oldFileSize) {
				oldFileFinished = true
			}
			newBytesRead, err := newFileReader.Read(newBytes)
			if err != nil {
				if err != io.EOF {
					return "", 0, fmt.Errorf("读取新版文件错误：%s", err)
				}
				newFileFinished = true
				break
			}
			newBytesReadSum += newBytesRead
			if newBytesReadSum >= int(newFileSize) {
				newFileFinished = true
			}
			diffFilePath := patch.GetPartDiffFileName(diffFileBasePath, partIndex)
			diffNewFilePath := patch.GetPartNewFileName(diffFileBasePath, partIndex)
			result, diffByteSize, err := DoBsDiffPart(oldBytes[:oldBytesRead], newBytes[:newBytesRead], diffFilePath, diffNewFilePath)
			if err != nil {
				return "", 0, fmt.Errorf("第 %d 块文件计算差异错误：%s", partIndex, err)
			}
			resultParts = append(resultParts, result)
			resultSize += diffByteSize
			partIndex++
		}
		if !newFileFinished {
			partFilePath := patch.GetPartNewFileName(diffFileBasePath, partIndex)
			bytesCopied, err := util.WriteAll(partFilePath, newFileReader)
			if err != nil {
				return "", 0, fmt.Errorf("写入第 %d 块文件错误：%s", partIndex, err)
			}
			if bytesCopied > 0 {
				resultParts = append(resultParts, patch.OperationTypeCopyNew)
				resultSize += int(bytesCopied)
			}
		}
		return strings.Join(resultParts, ","), resultSize, nil
	}
}

func getArgs() (oldDirAbsPath, newDirAbsPath, diffDirAbsPath string, bulkSize int, err error) {
	if len(os.Args) <= 3 {
		return "", "", "", 0, fmt.Errorf("调用参数数量不足\n\n%s", Help)
	}
	oldDir := os.Args[1]
	newDir := os.Args[2]
	diffDir := os.Args[3]
	if oldFileInfo, err := util.GetFileInfo(oldDir); err != nil {
		return "", "", "", 0, err
	} else if oldFileInfo == util.FileInfoResultNotExists {
		return "", "", "", 0, fmt.Errorf("旧版路径 %s 不存在", oldDir)
	} else if oldFileInfo == util.FileInfoResultExistFile {
		return "", "", "", 0, fmt.Errorf("旧版路径 %s 不是文件夹", oldDir)
	}
	if newFileInfo, err := util.GetFileInfo(newDir); err != nil {
		return "", "", "", 0, err
	} else if newFileInfo == util.FileInfoResultNotExists {
		return "", "", "", 0, fmt.Errorf("新版路径 %s 不存在", newDir)
	} else if newFileInfo == util.FileInfoResultExistFile {
		return "", "", "", 0, fmt.Errorf("新版路径 %s 不是文件夹", newDir)
	}
	if diffFileInfo, err := util.GetFileInfo(diffDir); err != nil {
		return "", "", "", 0, err
	} else if diffFileInfo == util.FileInfoResultExistFile {
		return "", "", "", 0, fmt.Errorf("输出差异路径 %s 不是文件夹", diffDir)
	}
	oldDirAbsPath, err = filepath.Abs(oldDir)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("获取旧版路径 %s 的绝对路径失败：%s", oldDir, err)
	}
	newDirAbsPath, err = filepath.Abs(newDir)
	if err != nil {
		return oldDirAbsPath, "", "", 0, fmt.Errorf("获取新版路径 %s 的绝对路径失败：%s", newDir, err)
	}
	diffDirAbsPath, err = filepath.Abs(diffDir)
	if err != nil {
		return oldDirAbsPath, newDirAbsPath, "", 0, fmt.Errorf("获取输出差异路径 %s 的绝对路径失败：%s", diffDir, err)
	}
	if len(os.Args) > 4 {
		bulkSize, err = strconv.Atoi(os.Args[4])
		if err != nil {
			return oldDirAbsPath, newDirAbsPath, diffDirAbsPath, 0, nil
		}
	} else {
		bulkSize = DefaultBulkSize
	}
	return oldDirAbsPath, newDirAbsPath, diffDirAbsPath, bulkSize, nil
}

func main() {
	oldDirAbsPath, newDirAbsPath, diffDirAbsPath, bulkSize, err := getArgs()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("旧版文件夹：", oldDirAbsPath)
	log.Println("新版文件夹：", newDirAbsPath)
	log.Println("输出差异文件夹：", diffDirAbsPath)
	if mkdirResult, err := util.MkdirIfNotExists(diffDirAbsPath); err != nil {
		log.Fatal("创建输出差异文件夹失败：", err)
	} else if mkdirResult == util.MkdirIfNotExistsResultExistsFile {
		log.Fatal("输出差异路径存在，但不是文件夹")
	} else if mkdirResult == util.MkdirIfNotExistsResultOK {
		log.Println("输出差异文件夹创建成功")
	}

	log.Println()
	log.Println("正在扫描旧版文件夹全部文件")
	oldFilesMD5, err := util.DirFilesMD5(oldDirAbsPath)

	log.Println()
	log.Println("正在扫描新版文件夹全部文件")
	newFilesMD5, err := util.DirFilesMD5(newDirAbsPath)

	log.Println()
	log.Println("正在列举未修改和新增文件")
	notModifiedFiles := make([]string, 0)
	addFiles := make([]string, 0)
	patchFiles := make([]string, 0)
	for fileName, fileMD5 := range newFilesMD5 {
		if oldFileMD5, ok := oldFilesMD5[fileName]; ok {
			if oldFileMD5 == fileMD5 {
				notModifiedFiles = append(notModifiedFiles, fileName)
			} else {
				patchFiles = append(patchFiles, fileName)
			}
		} else {
			addFiles = append(addFiles, fileName)
		}
	}

	log.Println()
	log.Println("差异文件列表")
	sort.Strings(notModifiedFiles)
	sort.Strings(addFiles)
	sort.Strings(patchFiles)
	for _, fileName := range notModifiedFiles {
		log.Println(" ", newFilesMD5[fileName], fileName)
	}
	for _, fileName := range addFiles {
		log.Println("+", newFilesMD5[fileName], fileName)
	}
	for _, fileName := range patchFiles {
		log.Println("*", newFilesMD5[fileName], fileName)
	}

	patchManifest := patch.NewPatchManifest(bulkSize)
	patchManifest.NewMd5 = make(map[string]string)
	patchManifest.OldMd5 = make(map[string]string)
	patchManifest.Patches = make(map[string]string)

	log.Println()
	log.Println("正在复制新文件")

	for _, fileName := range addFiles {
		newFilePath := newDirAbsPath + "\\" + fileName
		diffNewFilePath := diffDirAbsPath + "\\" + fileName
		diffNewFileDirPath := filepath.Dir(diffNewFilePath)
		if mkdirResult, err := util.MkdirIfNotExists(diffNewFileDirPath); err != nil {
			log.Fatal("创建文件夹失败：", err)
		} else if mkdirResult == util.MkdirIfNotExistsResultExistsFile {
			log.Fatal(diffNewFileDirPath, "路径存在，但不是文件夹")
		} else if mkdirResult == util.MkdirIfNotExistsResultOK {
			log.Println("创建文件夹成功")
		}
		if diffNewFileInfoResult, _ := util.GetFileInfo(diffNewFilePath); diffNewFileInfoResult == util.FileInfoResultExistFile {
			log.Println(fileName, "新文件已存在")
		} else if diffNewFileInfoResult == util.FileInfoResultExistDir {
			log.Fatal(diffNewFilePath, "路径已存在，但不是文件")
		} else {
			err := util.CopyFile(newFilePath, diffNewFilePath)
			if err != nil {
				log.Fatal(fileName, "复制新文件错误", err)
			}
			log.Println(fileName, "复制成功")
			patchManifest.Patches[fileName] = patch.OperationTypeCopyNew
			patchManifest.NewMd5[fileName] = newFilesMD5[fileName]
		}
	}

	log.Println()
	log.Println("正在计算文件差异")
	for _, fileName := range patchFiles {
		oldFilePath := oldDirAbsPath + "\\" + fileName
		newFilePath := newDirAbsPath + "\\" + fileName
		diffFileBasePath := diffDirAbsPath + "\\" + fileName
		diffNewFileDirPath := filepath.Dir(diffFileBasePath)
		if mkdirResult, err := util.MkdirIfNotExists(diffNewFileDirPath); err != nil {
			log.Fatal(diffNewFileDirPath, "创建文件夹失败：", err)
		} else if mkdirResult == util.MkdirIfNotExistsResultExistsFile {
			log.Fatal(diffNewFileDirPath, "文件夹存在，但不是文件夹")
		} else if mkdirResult == util.MkdirIfNotExistsResultOK {
			log.Println(diffNewFileDirPath, "创建文件夹成功")
		}
		log.Println(fileName, "正在计算差异")
		result, _, err := DoBsDiff(oldFilePath, newFilePath, diffFileBasePath, bulkSize)
		if err != nil {
			log.Fatal(fileName, "计算文件差异错误：", err)
		}
		log.Println(fileName, "差异计算完成")
		if result != patch.OperationTypeCopyNew {
			patchManifest.OldMd5[fileName] = oldFilesMD5[fileName]
		}
		patchManifest.Patches[fileName] = result
		patchManifest.NewMd5[fileName] = newFilesMD5[fileName]
	}

	log.Println()
	log.Println("正在生成补丁描述文件")
	for _, fileName := range notModifiedFiles {
		patchManifest.OldMd5[fileName] = oldFilesMD5[fileName]
		patchManifest.Patches[fileName] = "copy"
	}
	log.Println("补丁描述文件生成成功")

	data, err := json.Marshal(patchManifest)
	if err != nil {
		log.Fatal("补丁描述文件编码错误：", err)
	}
	patchJSONPath := filepath.Join(diffDirAbsPath, patch.ManifestFileName)
	err = ioutil.WriteFile(patchJSONPath, data, 0644)
	if err != nil {
		log.Fatal("输出补丁描述文件错误：", err)
	}
}
