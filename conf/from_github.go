package conf

import (
	"context"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"io"
)

var downloadGitHubContentFunc = downloadGitHubContent

func readConfigurationFromGitHub(owner, repo, filepath, token string) (*Configuration, error) {

	err := checkFileExtension(filepath)
	if err != nil {
		return nil, err
	}

	content, err := downloadGitHubContentFunc(owner, repo, filepath, token)
	if err != nil {
		return nil, err
	}

	defer content.Close()

	return readConfigurationContent(filepath, content)
}

func downloadGitHubContent(owner, repo, filepath, token string) (io.ReadCloser, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	return client.Repositories.DownloadContents(context.Background(), owner, repo, filepath, nil)
}