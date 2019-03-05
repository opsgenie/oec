package conf

import (
	"fmt"
	"github.com/opsgenie/ois/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestReadConfigurationFromJsonFile(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockJsonConfFileContent, ".json")
	assert.Nil(t, err)

	actualConf, err := readConfigurationFromFile(confPath)

	defer os.Remove(confPath)

	assert.Nil(t, err)
	assert.Equal(t, mockConf, actualConf,
		"Actual configuration was not equal to expected configuration.")
}

func TestReadConfigurationFromYamlFile(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockYamlConfFileContent, ".yaml")
	assert.Nil(t, err)

	actualConf, err := readConfigurationFromFile(confPath)

	defer os.Remove(confPath)

	assert.Nil(t, err)
	assert.Equal(t, mockConf, actualConf,
		"Actual configuration was not equal to expected configuration.")
}

func TestReadConfigurationFromInvalidFile(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockYamlConfFileContent, ".invalid")
	assert.Nil(t, err)

	_, err = readConfigurationFromFile(confPath)

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
