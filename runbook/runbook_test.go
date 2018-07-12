package runbook

import (
	"testing"
	"github.com/opsgenie/marid2/conf"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var testActionMappings = map[string]interface{}{
	"Create": map[string]interface{}{
		"filePath": "/path/to/runbook.bin",
		"source":   "local",
		"environmentVariables": map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
		},
	},
	"Close": map[string]interface{}{
		"source":       "github",
		"repoOwner":    "testAccount",
		"repoName":     "testRepo",
		"repoFilePath": "marid/testConfig.json",
		"repoToken":    "testtoken",
		"environmentVariables": map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
		},
	},
}

var executeRunbookFromGithubCalled = false
var executeRunbookFromLocalCalled = false

func mockExecuteRunbookFromGithub(runbookRepoOwner string, runbookRepoName string, runbookRepoFilePath string, runbookRepoToken string, environmentVariables map[string]interface{}) (string, string, error) {
	executeRunbookFromGithubCalled = true

	if len(runbookRepoOwner) <= 0 {
		return "", "", errors.New("runbookRepoOwner was empty.")
	}

	if len(runbookRepoName) <= 0 {
		return "", "", errors.New("runbookRepoName was empty.")
	}

	if len(runbookRepoFilePath) <= 0 {
		return "", "", errors.New("runbookRepoFilePath was empty.")
	}

	if len(runbookRepoToken) <= 0 {
		return "", "", errors.New("runbookRepoToken was empty.")
	}

	if len(environmentVariables) <= 0 {
		return "", "", errors.New("environmentVariables was empty.")
	}

	return "", "", nil
}

func mockExecuteRunbookFromLocal(executablePath string, environmentVariables map[string]interface{}) (string, string, error) {
	executeRunbookFromLocalCalled = true

	if len(executablePath) <= 0 {
		return "", "", errors.New("executablePath was empty.")
	}

	if len(environmentVariables) <= 0 {
		return "", "", errors.New("environmentVariables was empty.")
	}

	return "", "", nil
}

func TestExecuteRunbookGithub(t *testing.T) {
	conf.RunbookActionMapping = testActionMappings

	oldExecuteRunbookFromGithubFunction := executeRunbookFromGithubFunction
	defer func() { executeRunbookFromGithubFunction = oldExecuteRunbookFromGithubFunction }()
	executeRunbookFromGithubFunction = mockExecuteRunbookFromGithub
	cmdOut, cmdErr, err := ExecuteRunbook("Close")

	assert.NoError(t, err)
	assert.Equal(t, "", cmdOut)
	assert.Equal(t, "", cmdErr)
	assert.True(t, executeRunbookFromGithubCalled)
	executeRunbookFromGithubCalled = false
	assert.False(t, executeRunbookFromLocalCalled)
}

func TestExecuteRunbookLocal(t *testing.T) {
	conf.RunbookActionMapping = testActionMappings

	oldExecuteRunbookFromLocalFunction := executeRunbookFromLocalFunction
	defer func() { executeRunbookFromLocalFunction = oldExecuteRunbookFromLocalFunction }()
	executeRunbookFromLocalFunction = mockExecuteRunbookFromLocal
	cmdOut, cmdErr, err := ExecuteRunbook("Create")

	assert.NoError(t, err)
	assert.Equal(t, "", cmdOut)
	assert.Equal(t, "", cmdErr)
	assert.True(t, executeRunbookFromLocalCalled)
	executeRunbookFromLocalCalled = false
	assert.False(t, executeRunbookFromGithubCalled)
}
