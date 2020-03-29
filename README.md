# 文件夹 bsdiff

针对文件夹的 bsdiff

bsdiff 需要大量内存，为了避免大文件需要使用大量内存，使用了分块，分成 100 MB 的块进行对比。

同时由于还原使用的机器可能比计算差异使用的机器的 CPU 和内存大小都差很多，最好还是尽量减小文件，分成 100 MB 的小块，避免占用大量 CPU 和内存。 

## 使用

```bash
diff.exe 旧文件夹路径 新文件夹路径 差异文件夹路径 [文件分块大小]
patch.exe 旧文件夹路径 新文件夹路径 差异文件夹路径
```

## Build

```bash
go build ./cmd/diff
go build ./cmd/patch
```

## 关于 bsdiff

[bsdiff](http://www.daemonology.net/bsdiff/)

本项目使用 go-bsdiff，是 C 语言版本的速度 50% ~ 80% 左右

[Pure Go bsdiff and bspatch libraries and CLI tools.](https://github.com/gabstv/go-bsdiff)

## 补丁文件说明

```json
{
  "manifest_version": "0.1",
  "bulk_size": "104857600",
  "old_md5": {
  },
  "new_md5": {
  },
  "patches":  {
  }
}
```
