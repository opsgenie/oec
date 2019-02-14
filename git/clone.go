package git

import (
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"io/ioutil"
	"os"
)

var gitCloneMasterFunc = gitCloneMaster

const repositoryDirPrefix = "marid"

func CloneMaster(url, privateKeyFilepath, passPhrase string) (repositoryPath string, err error) {

	tmpDir, err := ioutil.TempDir("", repositoryDirPrefix)
	if err != nil {
		return "", err
	}

	err = gitCloneMasterFunc(tmpDir, url, privateKeyFilepath, passPhrase)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return tmpDir, nil
}

func gitCloneMaster(tmpDir, gitUrl, privateKeyFilepath, passPhrase string) error {

	options := &git.CloneOptions {
		URL:               	gitUrl,
		RecurseSubmodules: 	git.DefaultSubmoduleRecursionDepth, 	// todo max depth and master
		ReferenceName: 		plumbing.Master,
		SingleBranch:  		true,
	}

	if privateKeyFilepath != "" {

		auth, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, privateKeyFilepath, passPhrase)
		if err != nil {
			return err
		}

		options.Auth = auth
	}

	err := options.Validate()
	if err != nil {
		return err
	}

	_, err = git.PlainClone(tmpDir, false, options)

	return err
}
