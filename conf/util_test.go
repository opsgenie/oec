package conf

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestParseConfigurationJson(t *testing.T) {
	var directoryName = "testRepo"
	var tmpDir = os.TempDir() + string(os.PathSeparator) + directoryName
	var testConfPath = tmpDir + string(os.PathSeparator) + "maridTestConf.json"

	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	testFile, err := os.OpenFile(testConfPath, os.O_CREATE | os.O_WRONLY, 0755)

	if err != nil {
		t.Error("Error occurred while creating test config file. Error: " + err.Error())
	}

	testFile.WriteString("{\"apiKey\": \"apiKey\"}")
	testFile.Close()

	config, err := parseConfigurationFromFile(testConfPath)

	if err != nil {
		t.Error("Error occurred while parsing the conf file. Error: " + err.Error())
	}

	expectedConfig := &Configuration{ ApiKey: "apiKey" }

	assert.Equal(t, expectedConfig, config,
		"Actual configuration was not equal to expected configuration.")
}

func TestParseConfigurationYaml(t *testing.T) {
	var directoryName = "testRepo"
	var tmpDir = os.TempDir() + string(os.PathSeparator) + directoryName
	var testConfPath = tmpDir + string(os.PathSeparator) + "maridTestConf.yml"

	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	testFile, err := os.OpenFile(testConfPath, os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		t.Error("Error occurred while creating test config file. Error: " + err.Error())
	}

	testFile.WriteString("apiKey: apiKey\n")
	testFile.Close()

	config, err := parseConfigurationFromFile(testConfPath)

	if err != nil {
		t.Error("Error occurred while parsing the conf file. Error: " + err.Error())
	}

	expectedConfig := &Configuration{ ApiKey: "apiKey" }

	assert.Equal(t, expectedConfig, config,
		"Actual configuration was not equal to expected configuration.")
}