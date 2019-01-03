package runbook

import (
	"context"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"io/ioutil"
	"os"
	fpath "path/filepath"
)

var getRunbookFromGithubFunc = getRunbookFromGithub

func executeRunbookFromGithub(owner, repo, filepath, token string,
	args, environmentVariables []string) (string, string, error) {

	content, err := getRunbookFromGithubFunc(owner, repo, filepath, token)
	if err != nil {
		return "", "", err
	}

	filePath, err := writeContentToTemporaryFile(content, fpath.Ext(filepath))
	defer os.Remove(filePath)

	if err != nil {
		return "", "", err
	}

	err = os.Chmod(filePath, 0755)
	if err != nil {
		return "", "", err
	}

	return execute(filePath, args, environmentVariables)
}

func getRunbookFromGithub(owner, repo, filepath, token string) ([]byte, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	runbook, err := client.Repositories.DownloadContents(context.Background(), owner, repo, filepath, nil)
	if err != nil {
		return nil, err
	}
	defer runbook.Close()

	return ioutil.ReadAll(runbook)
}
