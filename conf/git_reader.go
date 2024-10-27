package conf

import (
	"os"
	fpath "path/filepath"

	"github.com/opsgenie/oec/git"
)

var cloneMasterFunc = git.CloneMaster

func readFileFromGit(url, privateKeyFilepath, passPhrase, filepath, branch string) (*Configuration, error) {

	err := checkFileExtension(filepath)
	if err != nil {
		return nil, err
	}

	repoFilepath, err := cloneMasterFunc(url, privateKeyFilepath, passPhrase, branch)
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(repoFilepath)

	filepath = fpath.Join(repoFilepath, filepath)

	return readFile(filepath)
}
