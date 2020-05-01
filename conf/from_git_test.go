package conf

import (
	"testing"

	"github.com/opsgenie/oec/git"
	"github.com/opsgenie/oec/util"
	"github.com/stretchr/testify/assert"
)

func TestReadConfigurationFromGit(t *testing.T) {

	defer func() { cloneMasterFunc = git.CloneMaster }()

	confPath, err := util.CreateTempTestFile(mockJsonConfFileContent, ".json")
	cloneMasterFunc = func(url, privateKeyFilepath, passPhrase string) (repositoryPath string, err error) {
		return "", nil
	}

	config, err := readConfigurationFromGit("", "", "", confPath)

	assert.Nil(t, err)
	assert.Equal(t, mockConf, config)
}
