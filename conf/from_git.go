package conf

import (
	"github.com/opsgenie/marid2/git"
	"os"
	"path/filepath"
)

var cloneRepositoryFunc = git.CloneRepository

func readConfigurationFromGit(url, privateKeyFilepath, passPhrase, confFilepath string) (*Configuration, error) {

	err := checkFileExtension(confFilepath)
	if err != nil {
		return nil, err
	}

	repoFilepath, err := cloneRepositoryFunc(url, privateKeyFilepath, passPhrase)
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(repoFilepath)

	confFilepath = filepath.Join(repoFilepath, confFilepath)

	return readConfigurationFromFile(confFilepath)
}