package conf

import (
	"github.com/opsgenie/oec/git"
	"os"
	"path/filepath"
)

var cloneMasterFunc = git.CloneMaster

func readConfigurationFromGit(url, privateKeyFilepath, passPhrase, confFilepath string) (*Configuration, error) {

	err := checkFileExtension(confFilepath)
	if err != nil {
		return nil, err
	}

	repoFilepath, err := cloneMasterFunc(url, privateKeyFilepath, passPhrase)
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(repoFilepath)

	confFilepath = filepath.Join(repoFilepath, confFilepath)

	return readConfigurationFromFile(confFilepath)
}
