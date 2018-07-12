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

var gitCloneMethod = gitClone

func readConfigurationFromGit(url string, confPath string, privateKeyFilePath string) (map[string]interface{}, error) {
	var tmpDir = os.TempDir()

	err := os.MkdirAll(tmpDir, 0755)

	if err != nil {
		return nil, err
	}

	directoryName, err := parseDirectoryNameFromUrl(url)
	defer os.RemoveAll(tmpDir + string(os.PathSeparator) + directoryName)

	if err != nil {
		return nil, err
	}

	err = gitCloneMethod(url, privateKeyFilePath)

	configuration, err := parseConfiguration(tmpDir + string(os.PathSeparator) + directoryName +
		string(os.PathSeparator) + confPath)

	if err != nil {
		return nil, err
	}

	return configuration, nil
}

func gitClone(gitUrl string, privateKeyFilePath string) error {
	cloneOptions, err := getCloneOptions(gitUrl, privateKeyFilePath)

	if err != nil {
		return err
	}

	_, err = git.PlainClone(""+string(os.PathSeparator)+"", false, &cloneOptions)

	return err
}
func getCloneOptions(gitUrl, privateKeyFilePath string) (git.CloneOptions, error) {
	file, err := ioutil.ReadFile(privateKeyFilePath)

	if err != nil {
		return git.CloneOptions{}, err
	}

	signer, err := ssh.ParsePrivateKey(file)

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