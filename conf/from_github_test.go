package conf

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestReadConfigurationFromGithub(t *testing.T) {

	oldDownloadFunc := downloadGitHubContent
	defer func() { downloadGitHubContentFunc = oldDownloadFunc}()
	downloadGitHubContentFunc = func(owner, repo, filepath, token string) (io.ReadCloser, error) {
		res := http.Response{
			Body: ioutil.NopCloser(bytes.NewBuffer(mockConfFileContent)),
		}
		return res.Body, nil
	}

	config, err := readConfigurationFromGitHub("owner", "repo", "dummy.json", "token")

	assert.Nil(t, err)
	assert.Equal(t, mockConf, config)

}

func TestReadConfigurationFromGithubWithInvalidFileExtension(t *testing.T) {

	_, err := readConfigurationFromGitHub("owner", "repo", "dummy.x", "token")

	expectedErr := errors.Errorf(unknownFileExtErrMessage, ".x")
	assert.EqualError(t, err, expectedErr.Error())
}