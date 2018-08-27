package runbook

import (
	"github.com/opsgenie/marid2/conf"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestSendResultToOpsGenieWithParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if r.Method != "POST" {
			t.Errorf("Expected ‘POST’ request, got ‘%s’", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded; charset=UTF-8" {
			t.Errorf("Expected request to have ‘Content-Type=application/x-www-form-urlencoded; charset=UTF-8’, got: ‘%s’", contentType)
		}

		r.ParseForm()

		alertAction := r.Form.Get("mappedAction")
		if alertAction != "testAction" {
			t.Errorf("Expected request to have ‘mappedAction=testAction’, got: ‘%s’", alertAction)
		}

		alertId := r.Form.Get("alertId")
		if alertId != "testAlert" {
			t.Errorf("Expected request to have ‘alertId=testAlert’, got: ‘%s’", alertId)
		}

		success := r.Form.Get("success")
		if success != "true" {
			t.Errorf("Expected request to have ‘success=true’, got: ‘%s’", success)
		}

		apiKey := r.Form.Get("apiKey")
		if apiKey != "testKey" {
			t.Errorf("Expected request to have ‘apiKey=testKey’, got: ‘%s’", apiKey)
		}

	}))
	defer ts.Close()

	params := make(map[string]interface{})

	mappedAction := make(map[string]interface{})
	mappedAction["name"] = "testAction"

	params["mappedActionV2"] = mappedAction
	params["alertId"] = "testAlert"

	conf.Configuration = make(map[string]interface{})
	conf.Configuration["apiKey"] = "testKey"
	conf.Configuration["opsgenieApiUrl"] = ts.URL // To do: finalize it before releasing

	sendResultToOpsGenie("createAlert", "123123", params, "")
}

func TestSendResultToOpsGenieWithoutParamsAndWithFailureMsg(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if r.Method != "POST" {
			t.Errorf("Expected ‘POST’ request, got ‘%s’", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded; charset=UTF-8" {
			t.Errorf("Expected request to have ‘Content-Type=application/x-www-form-urlencoded; charset=UTF-8’, got: ‘%s’", contentType)
		}

		r.ParseForm()

		alertAction := r.Form.Get("alertAction")
		if alertAction != "createAlert" {
			t.Errorf("Expected request to have ‘alertAction=createAlert’, got: ‘%s’", alertAction)
		}

		failureMsg := r.Form.Get("failureMessage")
		if failureMsg != "testFailureMessage" {
			t.Errorf("Expected request to have ‘failureMessage=testFailureMessage’, got: ‘%s’", failureMsg)
		}

		success := r.Form.Get("success")
		if success != "false" {
			t.Errorf("Expected request to have ‘success=false’, got: ‘%s’", success)
		}

		apiKey := r.Form.Get("apiKey")
		if apiKey != "testKey" {
			t.Errorf("Expected request to have ‘apiKey=testKey’, got: ‘%s’", apiKey)
		}

	}))
	defer ts.Close()

	conf.Configuration = make(map[string]interface{})
	conf.Configuration["apiKey"] = "testKey"
	conf.Configuration["opsgenieApiUrl"] = ts.URL // To do: finalize it before releasing

	sendResultToOpsGenie("createAlert", "123123", nil, "testFailureMessage")
}
