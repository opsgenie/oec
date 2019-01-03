package runbook

import (
	"errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

var tempDir = os.TempDir() + string(os.PathSeparator)

func writeContentToTemporaryFile(content []byte, fileExtension string) (string, error) {

	tempFile, err := ioutil.TempFile(tempDir, "*" + fileExtension)
	if err != nil {
		logrus.Error(err)
	}

	defer tempFile.Close()

	if _, err := tempFile.Write(content); err != nil {
		logrus.Error(err)
	}

	return tempFile.Name(), nil
}

func createTestScriptFile(content []byte, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE | os.O_WRONLY, 0755)

	if err != nil {
		return errors.New("Error occurred while creating test script file. Error: " + err.Error())
	}

	if _, err = file.Write(content); err != nil {
		return err
	}

	file.Close()

	return nil
}
