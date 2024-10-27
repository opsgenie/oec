package conf

import (
	"fmt"
	"os"
	"testing"

	"github.com/opsgenie/oec/util"
	"github.com/stretchr/testify/assert"
)

func TestReadJsonFile(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockJsonFileContent, ".json")
	assert.Nil(t, err)

	actualConf, err := readFile(confPath)

	defer os.Remove(confPath)

	assert.Nil(t, err)
	assert.Equal(t, mockConf, actualConf,
		"Actual configuration was not equal to expected configuration.")
}

func TestReadYamlFile(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockYamlFileContent, ".yaml")
	assert.Nil(t, err)

	actualConf, err := readFile(confPath)

	defer os.Remove(confPath)

	assert.Nil(t, err)
	assert.Equal(t, mockConf, actualConf,
		"Actual configuration was not equal to expected configuration.")
}

func TestReadInvalidFile(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockYamlFileContent, ".invalid")
	assert.Nil(t, err)

	_, err = readFile(confPath)

	defer os.Remove(confPath)

	assert.NotNil(t, err)
	expectedErrMsg := fmt.Sprintf(unknownFileExtErrMessage, ".invalid")
	assert.EqualError(t, err, expectedErrMsg)
}

func TestCheckFileExtensionInvalidExt(t *testing.T) {

	err := checkFileExtension("/dummy.invalid")

	expectedErrMsg := fmt.Sprintf(unknownFileExtErrMessage, ".invalid")
	assert.EqualError(t, err, expectedErrMsg)
}

func TestCheckFileExtension(t *testing.T) {

	err := checkFileExtension("/dummy.json")
	assert.Nil(t, err)

	err = checkFileExtension("/dummy.yml")
	assert.Nil(t, err)

	err = checkFileExtension("/dummy.yaml")
	assert.Nil(t, err)
}
