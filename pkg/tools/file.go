package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

//IsExist return true if file exists
func IsExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil || os.IsExist(err)
}

//DeCompressTgz
func DeCompressTgz(tarFile, destPath string) error {
	file, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	for {
		next, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fileName := destPath + "/" + next.Name
		err = createFile(fileName, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

func createFile(fileName string, reader *tar.Reader) error {
	err := os.MkdirAll(string([]rune(fileName)[0:strings.LastIndex(fileName, "/")]), 0755)
	if err != nil {
		return err
	}
	if fileName[len(fileName)-1:] == "/" {
		return nil
	}
	fw, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer fw.Close()
	_, err = io.Copy(fw, reader)
	return err
}

// 校验 md5
func CheckMd5(filePath, md5sum string) (bool, error) {
	sum, err := Md5sum(filePath)
	if err != nil {
		return false, err
	}
	b := sum == md5sum
	if b {
		return b, nil
	}
	return false, errors.New("md5 not equal")
}

// 获取文件 md5
func Md5sum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func Md5sumData(data interface{}) (string, error) {
	if data == nil {
		return "", fmt.Errorf("md5 data is nill")
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	sum := md5.Sum(bytes)
	return fmt.Sprintf("%x", sum), nil
}

func CompressByGzip(body string) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte(body))
	if err != nil {
		return []byte{}, err
	}
	zw.Close()
	return buf.Bytes(), nil
}

func DecompressByGzip(body []byte) (string, error) {
	zr, err := gzip.NewReader(bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, zr); err != nil {
		return "", err
	}
	zr.Close()
	return buf.String(), nil
}
