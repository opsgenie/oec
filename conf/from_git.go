package conf

import (
	"gopkg.in/src-d/go-git.v4"
	"strings"
	"github.com/pkg/errors"
	"os"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

var gitCloneMethod = gitClone

func readConfigurationFromGit(url string, username string, password string) (map[string]string, error) {
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

	err = gitCloneMethod(url, username, password)

	configuration, err := parseConfiguration(tmpDir + string(os.PathSeparator) + directoryName +
		string(os.PathSeparator) + "marid.conf")

	if err != nil {
		return nil, err
	}

	return configuration, nil
}

func gitClone(url string, username string, password string) error {
	_, err := git.PlainClone("" + string(os.PathSeparator) + "", false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
	})

	if err != nil {
		return err
	} else {
		return nil
	}
}

func parseDirectoryNameFromUrl(url string) (string, error) {
	urlWithoutExtension := strings.TrimRight(url, ".git")
	lastIndex := strings.LastIndex(urlWithoutExtension, "/")

	if lastIndex == -1 {
		return "", errors.New(url + " is not a valid Git URL.")
	}

	return urlWithoutExtension[lastIndex+1:], nil
}