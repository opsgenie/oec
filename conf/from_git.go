package conf

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	goGitSsh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var gitCloneFunc = gitClone

func readConfigurationFromGit(url string, confPath string, privateKeyFilePath string, passPhrase string) (*Configuration, error) {

	err := os.MkdirAll(os.TempDir(), 0755)

	if err != nil {
		return nil, err
	}

	directoryName, err := parseDirectoryNameFromUrl(url)
	if err != nil {
		return nil, err
	}

	tmpDir := os.TempDir() + string(os.PathSeparator) + directoryName

	os.RemoveAll(tmpDir)	// todo validate path traversal
	defer os.RemoveAll(tmpDir)

	err = gitCloneFunc(tmpDir, url, privateKeyFilePath, passPhrase)

	if err != nil {
		return nil, err
	}

	return parseConfiguration(tmpDir + string(os.PathSeparator) + confPath)
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
	if !strings.HasSuffix(url, ".git") {
		return "", errors.New(url + " is not a valid Git URL.")
	}

	urlWithoutExtension := strings.TrimRight(url, ".git")
	lastIndex := strings.LastIndex(urlWithoutExtension, "/")

	if lastIndex == -1 {
		return "", errors.New(url + " is not a valid Git URL.")
	}

	dirName := filepath.Clean(urlWithoutExtension[lastIndex:])

	if dirName == "" || dirName == "/" || dirName == "." {
		return "", errors.New(url + " is not a valid Git URL.")
	}

	return dirName, nil
}