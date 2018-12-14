package conf

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
)

func TestParseConfigurationJson(t *testing.T) {
	var directoryName = "testRepo"
	var tmpDir = os.TempDir()
	var testConfPath = tmpDir + string(os.PathSeparator) + directoryName + string(os.PathSeparator) +
		"maridConf.json"

	os.MkdirAll(tmpDir+string(os.PathSeparator)+directoryName, 0755)
	defer os.RemoveAll(tmpDir + string(os.PathSeparator) + directoryName)

	testFile, err := os.OpenFile(testConfPath, os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		t.Error("Error occurred while creating test config file. Error: " + err.Error())
	}

	testFile.WriteString("{\"apiKey\": \"apiKey\"}")
	testFile.Close()

	config, err := parseConfiguration(testConfPath)

	if err != nil {
		t.Error("Error occurred while parsing the conf file. Error: " + err.Error())
	}

	expectedConfig := &Configuration{ ApiKey: "apiKey" }

	assert.Equal(t, expectedConfig, config,
		"Actual configuration was not equal to expected configuration.")
}

func TestParseConfigurationYaml(t *testing.T) {
	var directoryName = "testRepo"
	var tmpDir = os.TempDir()
	var testConfPath = tmpDir + string(os.PathSeparator) + directoryName + string(os.PathSeparator) +
		"maridConf.yml"

	os.MkdirAll(tmpDir+string(os.PathSeparator)+directoryName, 0755)
	defer os.RemoveAll(tmpDir + string(os.PathSeparator) + directoryName)

	testFile, err := os.OpenFile(testConfPath, os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		t.Error("Error occurred while creating test config file. Error: " + err.Error())
	}

	testFile.WriteString("apiKey: apiKey\n")
	testFile.Close()

	config, err := parseConfiguration(testConfPath)

	if err != nil {
		t.Error("Error occurred while parsing the conf file. Error: " + err.Error())
	}

	expectedConfig := &Configuration{ ApiKey: "apiKey" }

	assert.Equal(t, expectedConfig, config,
		"Actual configuration was not equal to expected configuration.")
}