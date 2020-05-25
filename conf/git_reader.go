package conf

import (
	"github.com/opsgenie/oec/git"
	"os"
	fpath "path/filepath"
)

var cloneMasterFunc = git.CloneMaster

func readFileFromGit(url, privateKeyFilepath, passPhrase, filepath string) (*Configuration, error) {

	err := checkFileExtension(filepath)
	if err != nil {
		return nil, err
	}

	repoFilepath, err := cloneMasterFunc(url, privateKeyFilepath, passPhrase)
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(repoFilepath)

	filepath = fpath.Join(repoFilepath, filepath)

	return readFile(filepath)
}
