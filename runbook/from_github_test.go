package runbook

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"os"
)

var getRunbookFromGithubCalled = false

func mockGetRunbookFromGithub(owner string, repo string, filepath string, token string) (string, error) {
	getRunbookFromGithubCalled = true

	return "echo \"testcontent\"\n", nil
}

func TestExecuteRunbookFromGithub(t *testing.T) {
	var testScriptPath = os.TempDir() + string(os.PathSeparator) + "test.sh"

	oldGetRunbookFromGithubMethod := getRunbookFromGithubFunction
	defer func() { getRunbookFromGithubFunction = oldGetRunbookFromGithubMethod }()
	getRunbookFromGithubFunction = mockGetRunbookFromGithub

	cmdOut, cmdErr, err := executeRunbookFromGithub("testOwner", "testRepo", "test.sh",
		"testToken", nil)

	assert.NoError(t, err, "Error from execute operation was not empty.")
	assert.Equal(t, "testcontent\n", cmdOut, "Output stream was not equal to expected.")
	assert.Equal(t, "", cmdErr, "Error stream from executed file was not empty.")

	if _, err := os.Stat(testScriptPath); !os.IsNotExist(err) {
		t.Error("Test script was not deleted after execution.")
	}

	assert.True(t, getRunbookFromGithubCalled, "getRunbookFromGithub was not called.")
	getRunbookFromGithubCalled = false
}
