package conf

import (
	"os"
	"testing"

	"github.com/opsgenie/oec/util"
	"github.com/stretchr/testify/assert"
)

func TestReadConfigurationFromLocal(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockJsonConfFileContent, ".json")
	assert.Nil(t, err)

	actualConf, _ := readConfigurationFromLocal(confPath)

	defer os.Remove(confPath)

	assert.Equal(t, mockConf, actualConf,
		"Actual configuration was not equal to expected configuration.")
}
