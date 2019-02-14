package conf

import (
	"github.com/opsgenie/marid2/git"
	"github.com/opsgenie/marid2/util"
	"github.com/stretchr/testify/assert"
	"testing"
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