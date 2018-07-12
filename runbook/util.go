package runbook

import (
	"github.com/google/uuid"
	"strings"
	"os"
	"fmt"
	"errors"
)

func writeContentToTemporaryFile(content string, fileName string) (string, error) {
	tmpDir := os.TempDir()
	fullPath := tmpDir + string(os.PathSeparator) + fileName

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		var newFileName string

		for {
			newFileName, err = appendUniqueRandomPostfixToFileName(fileName)

			if err != nil {
				return "", err
			}

			fullPath = tmpDir + string(os.PathSeparator) + newFileName

			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				break
			}
		}
	}

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, 0766)

	if err != nil {
		return "", nil
	}

	file.WriteString(content)
	file.Close()

	return fullPath, nil
}

func appendUniqueRandomPostfixToFileName(fileName string) (string, error) {
	randUUID, err := uuid.NewRandom()

	if err != nil {
		return "", err
	}

	newFileName := fileName

	if dotIndex := strings.LastIndex(newFileName, "."); dotIndex != -1 {
		newFileName = newFileName[0:dotIndex] + "-" + randUUID.String() + newFileName[dotIndex:]
	} else {
		newFileName = newFileName + "-" + randUUID.String()
	}

	return newFileName, nil
}

func convertMapToArray(sourceMap map[string]interface{}) []string {
	destinationArray := make([]string, 0)

	for key, value := range sourceMap {
		destinationArray = append(destinationArray, fmt.Sprintf("%s=%s", key, value))
	}

	return destinationArray
}

func createTestScriptFile(content string, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		return errors.New("Error occurred while creating test script file. Error: " + err.Error())
	}

	file.WriteString(content)
	file.Close()

	return nil
}
