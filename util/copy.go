package util

import (
	"io"
	"os"
)

const (
	CopyBufferSize = 1024 * 1204
)

// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
// https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang/21061062#21061062
func CopyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, CopyBufferSize)
	_, err = io.CopyBuffer(out, in, buf)
	if err != nil {
		return err
	}
	return out.Close()
}

func WriteAll(path string, r io.Reader) (int64, error) {
	w, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	defer w.Close()

	buf := make([]byte, CopyBufferSize)
	n, err := io.CopyBuffer(w, r, buf)
	if err != nil {
		return 0, err
	}

	return n, nil
}
