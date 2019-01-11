package git

import (
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"io/ioutil"
	"os"
)

var gitCloneFunc = gitClone

const repositoryDirPrefix = "marid"

func CloneRepository(url, privateKeyFilepath, passPhrase string) (repositoryPath string, err error) {

	tmpDir, err := ioutil.TempDir("", repositoryDirPrefix)
	if err != nil {
		return "", err
	}

	err = gitCloneFunc(tmpDir, url, privateKeyFilepath, passPhrase)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return tmpDir, nil
}

func gitClone(tmpDir, gitUrl, privateKeyFilepath, passPhrase string) error {

	cloneOptions := &git.CloneOptions {
		URL:               gitUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth, 	// todo max depth and master
	}

	if privateKeyFilepath != "" {

		auth, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, privateKeyFilepath, passPhrase)
		if err != nil {
			return err
		}

		cloneOptions.Auth = auth
	}

	err := cloneOptions.Validate()
	if err != nil {
		return err
	}

	_, err = git.PlainClone(tmpDir, false, cloneOptions)

	return err
}
