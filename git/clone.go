package git

import (
	"io/ioutil"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

var cloneMasterFunc = cloneMaster

const repositoryDirPrefix = "oec"

func CloneMaster(url, privateKeyFilepath, passPhrase, branch string) (repositoryPath string, err error) {

	tmpDir, err := ioutil.TempDir("", repositoryDirPrefix)
	if err != nil {
		return "", err
	}

	err = cloneMasterFunc(tmpDir, url, privateKeyFilepath, passPhrase, branch)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return tmpDir, nil
}

func cloneMaster(tmpDir, gitUrl, privateKeyFilepath, passPhrase, branch string) error {

	if branch == "" {
		branch = "master"
	}

	options := &git.CloneOptions{
		URL:               gitUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth, // todo restrict max depth
		ReferenceName:     plumbing.NewBranchReferenceName(branch),
		SingleBranch:      true,
	}

	if privateKeyFilepath != "" {

		auth, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, privateKeyFilepath, passPhrase)
		if err != nil {
			return err
		}

		options.Auth = auth
	}

	_, err := git.PlainClone(tmpDir, false, options)

	return err
}
