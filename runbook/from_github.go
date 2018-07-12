package runbook

import (
	"golang.org/x/oauth2"
	"github.com/google/go-github/github"
	"io/ioutil"
	"context"
	"path"
	"os"
)

var getRunbookFromGithubFunction = getRunbookFromGithub

func executeRunbookFromGithub(runbookRepoOwner string, runbookRepoName string, runbookRepoFilePath string,
	runbookRepoToken string, environmentVariables map[string]interface{}) (string, string, error) {
	content, err := getRunbookFromGithubFunction(runbookRepoOwner, runbookRepoName, runbookRepoFilePath, runbookRepoToken)

	if err != nil {
		return "", "", err
	}

	filePath, err := writeContentToTemporaryFile(content, path.Base(runbookRepoFilePath))
	defer os.Remove(filePath)

	if err != nil {
		return "", "", err
	}

	err = os.Chmod(filePath, 0755)

	if err != nil {
		return "", "", err
	}

	return execute(filePath, nil, environmentVariables)
}

func getRunbookFromGithub(owner string, repo string, filepath string, token string) (string, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	runbook, _ := client.Repositories.DownloadContents(context.Background(), owner, repo, filepath, nil)
	defer runbook.Close()
	bytes, err := ioutil.ReadAll(runbook)

	return string(bytes), err
}
