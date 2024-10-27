package conf

import (
	"testing"

	"github.com/opsgenie/oec/git"
	"github.com/opsgenie/oec/util"
	"github.com/stretchr/testify/assert"
)

func TestReadFileFromGit(t *testing.T) {

	defer func() { cloneMasterFunc = git.CloneMaster }()

	confPath, err := util.CreateTempTestFile(mockJsonFileContent, ".json")
	cloneMasterFunc = func(url, privateKeyFilepath, passPhrase, branch string) (repositoryPath string, err error) {
		return "", nil
	}

	config, err := readFileFromGit("", "", "", confPath, "")

	assert.Nil(t, err)
	assert.Equal(t, mockConf, config)
}
