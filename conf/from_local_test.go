package conf

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
)

func TestReadConfigurationFromLocal(t *testing.T) {
	homePath, err := getHomePath()
	confPath := homePath + string(os.PathSeparator) + ".opsgenie" +
		string(os.PathSeparator) + "maridConf.json"

	if err != nil {
		t.Error("Error occurred during obtaining user's home path. Error: " + err.Error())
	}

	if _, err := os.Stat(homePath + string(os.PathSeparator) + ".opsgenie"); os.IsNotExist(err) {
		os.Mkdir(homePath + string(os.PathSeparator) + ".opsgenie", 0755)
	}

	testConfFile, err := os.OpenFile(confPath, os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		t.Error("Error occurred during writing test Marid configuration file. Error: " + err.Error())
	}

	testConfFile.WriteString(mockConfFileContent)
	testConfFile.Close()
	configurationFromLocal, _ := readConfigurationFromLocal(confPath)

	defer os.Remove(confPath)

	assert.Equal(t, mockConf, configurationFromLocal,
		"Actual config and expected config are not the same.")
}
