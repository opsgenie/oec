package conf

import (
	"gopkg.in/src-d/go-git.v4"
	"strings"
	"github.com/pkg/errors"
	"os"
	"io/ioutil"
	"golang.org/x/crypto/ssh"
	goGitSsh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

var gitCloneFunc = gitClone

func readConfigurationFromGit(url string, confPath string, privateKeyFilePath string, passPhrase string) (*Configuration, error) {
	var tmpDir = os.TempDir()

	err := os.MkdirAll(tmpDir, 0755)

	if err != nil {
		return nil, err
	}

	directoryName, err := parseDirectoryNameFromUrl(url)
	os.RemoveAll(tmpDir + string(os.PathSeparator) + directoryName)	// todo validate path traversal
	defer os.RemoveAll(tmpDir + string(os.PathSeparator) + directoryName)

	if err != nil {
		return nil, err
	}

	err = gitCloneFunc(tmpDir + string(os.PathSeparator) + directoryName, url, privateKeyFilePath, passPhrase)

	if err != nil {
		return nil, err
	}

	return parseConfiguration(tmpDir + string(os.PathSeparator) + directoryName + string(os.PathSeparator) + confPath)
}

func gitClone(tmpDir string, gitUrl string, privateKeyFilePath string, passPhrase string) error {
	cloneOptions, err := getCloneOptions(gitUrl, privateKeyFilePath, passPhrase)

	if err != nil {
		return err
	}

	_, err = git.PlainClone(tmpDir, false, &cloneOptions)

	return err
}

func getCloneOptions(gitUrl, privateKeyFilePath string, passPhrase string) (git.CloneOptions, error) {
	file, err := ioutil.ReadFile(privateKeyFilePath)

	if err != nil {
		return git.CloneOptions{}, err
	}

	signer, err := ssh.ParsePrivateKeyWithPassphrase(file, []byte(passPhrase))

	if err != nil {
		return git.CloneOptions{}, err
	}

	auth := goGitSsh.PublicKeys{User: "git", Signer: signer}

	return git.CloneOptions{
		URL:               gitUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              &auth,
	}, nil
}

func parseDirectoryNameFromUrl(url string) (string, error) {
	urlWithoutExtension := strings.TrimRight(url, ".git")
	lastIndex := strings.LastIndex(urlWithoutExtension, "/")

	if lastIndex == -1 {
		return "", errors.New(url + " is not a valid Git URL.")
	}

	return urlWithoutExtension[lastIndex+1:], nil
}