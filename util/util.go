package util

import (
	"io/ioutil"
)

func Min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

func CreateTempTestFile(content []byte, fileExtension string) (string, error) {

	tempFile, err := ioutil.TempFile("", "*" + fileExtension)
	if err != nil {
		return "", err
	}

	defer tempFile.Close()

	if _, err := tempFile.Write(content); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}
