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

	testFile.WriteString("{\"tk1\": \"tv1\",\"tk2\": \"tv2\", \"emre\": \"cicek\"}")
	testFile.Close()

	config, err := parseConfiguration(testConfPath)

	if err != nil {
		t.Error("Error occurred while parsing the conf file. Error: " + err.Error())
	}

	expectedConfig := map[string]interface{}{
		"tk1": "tv1",
		"tk2": "tv2",
		"emre": "cicek",
	}

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

	testFile.WriteString("tk1: tv1\ntk2: tv2\nemre: cicek\n")
	testFile.Close()

	config, err := parseConfiguration(testConfPath)

	if err != nil {
		t.Error("Error occurred while parsing the conf file. Error: " + err.Error())
	}

	expectedConfig := map[string]interface{}{
		"tk1":  "tv1",
		"tk2":  "tv2",
		"emre": "cicek",
	}

	assert.Equal(t, expectedConfig, config,
		"Actual configuration was not equal to expected configuration.")
}

func TestCloneMap(t *testing.T){
	expectedMap := map[string]interface{}{
		"k1": "v1",
		"k2": "v2",
	}

	clonedMap, err := cloneMap(expectedMap)
	assert.NoError(t, err, "Error occurred during map clone.")
	assert.True(t, assert.ObjectsAreEqualValues(expectedMap, clonedMap),
		"Original map and cloned map are not the same.")
}