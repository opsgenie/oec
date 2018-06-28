package conf

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
)

func TestParseConfiguration(t *testing.T) {
	var directoryName = "testRepo"
	var tmpDir = os.TempDir()

	os.MkdirAll(tmpDir+string(os.PathSeparator)+directoryName, 0755)
	defer os.RemoveAll(tmpDir + string(os.PathSeparator) + directoryName)

	testFile, err := os.OpenFile(tmpDir+string(os.PathSeparator)+directoryName+string(os.PathSeparator)+
		"marid.conf", os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		t.Error("Error occurred while creating test config file. Error: " + err.Error())
	}

	testFile.WriteString("tk1=tv1\ntk2=tv2\nemre=cicek")
	testFile.Close()

	config, err := parseConfiguration(tmpDir+string(os.PathSeparator)+directoryName+string(os.PathSeparator)+
		"marid.conf")

	expectedConfig := map[string]string{
		"tk1": "tv1",
		"tk2": "tv2",
		"emre": "cicek",
	}

	assert.True(t, assert.ObjectsAreEqual(expectedConfig, config),
		"Actual configuration was not equal to expected configuration.")
}

func TestCloneMap(t *testing.T){
	expectedMap := map[string]string {
		"k1": "v1",
		"k2": "v2",
	}

	clonedMap, err := cloneStringMap(expectedMap)
	assert.NoError(t, err, "Error occurred during map clone.")
	assert.True(t, assert.ObjectsAreEqual(expectedMap, clonedMap),
		"Original map and cloned map are not the same.")
}