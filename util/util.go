package util

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func Min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

func CreateTempTestFile(content []byte, fileExtension string) (string, error) {

	tempFile, err := ioutil.TempFile("", "*"+fileExtension)
	if err != nil {
		return "", err
	}

	defer tempFile.Close()

	if _, err := tempFile.Write(content); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func ChmodRecursively(path string, mode os.FileMode) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, mode)
	})
}

func CheckLogFile(logger *lumberjack.Logger, interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
			if _, err := os.Stat(logger.Filename); os.IsNotExist(err) {
				logrus.Warnf("Failed to open OEC log file: %v. New file will be created.", err)
				if err = logger.Rotate(); err != nil {
					logrus.Warn(err)
				} else {
					logrus.Warnf("New OEC log file[%s] is created, previous one might be removed accidentally.", logger.Filename)
				}
				break
			}
		}
	}
}
