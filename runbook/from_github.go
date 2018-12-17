package runbook

import (
	"context"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"io/ioutil"
	"os"
	"path"
)

var getRunbookFromGithubFunc = getRunbookFromGithub

func executeRunbookFromGithub(repoOwner string, repoName string, repoFilePath string,
	repoToken string, args []string, environmentVariables []string) (string, string, error) {

	content, err := getRunbookFromGithubFunc(repoOwner, repoName, repoFilePath, repoToken)

	if err != nil {
		return "", "", err
	}

	filePath, err := writeContentToTemporaryFile(content, path.Base(repoFilePath))
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

func getRunbookFromGithub(owner string, repo string, filepath string, token string) ([]byte, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	runbook, _ := client.Repositories.DownloadContents(context.Background(), owner, repo, filepath, nil)
	defer runbook.Close()

	return ioutil.ReadAll(runbook)
}
