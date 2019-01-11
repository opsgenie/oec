package conf

import (
	"github.com/opsgenie/marid2/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestReadConfigurationFromLocal(t *testing.T) {

	confPath, err := util.CreateTempTestFile(mockJsonConfFileContent, ".json")
	assert.Nil(t, err)

	actualConf, _ := readConfigurationFromLocal(confPath)

	defer os.Remove(confPath)

	assert.Equal(t, mockConf, actualConf,
		"Actual configuration was not equal to expected configuration.")
}
