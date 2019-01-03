package runbook

import (
	"encoding/json"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/retryer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockActionMappings = (map[conf.ActionName]conf.MappedAction)(conf.ActionMappings{
	"Create" : conf.MappedAction{
		Source:               "local",
		FilePath:             "/path/to/runbook.bin",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
	"Close" : conf.MappedAction{
		Source:               "github",
		RepoName:             "testRepo",
		RepoOwner:            "testAccount",
		RepoToken:            "testToken",
		RepoFilePath:         "marid/testConfig.json",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
})

var executeRunbookFromGithubCalled = false
var executeRunbookFromLocalCalled = false

func mockExecuteRunbookFromGithub(owner, repo, filepath, token string, args, environmentVariables []string) (string, string, error) {
	executeRunbookFromGithubCalled = true

	if len(owner) <= 0 {
		return "", "", errors.New("owner was empty.")
	}

	if len(repo) <= 0 {
		return "", "", errors.New("repo was empty.")
	}

	if len(filepath) <= 0 {
		return "", "", errors.New("filepath was empty.")
	}

	if len(token) <= 0 {
		return "", "", errors.New("token was empty.")
	}

	if len(environmentVariables) <= 0 {
		return "", "", errors.New("environmentVariables was empty.")
	}

	return "", "", nil
}

func mockExecuteRunbookFromLocal(executablePath string, args, environmentVariables []string) (string, string, error) {
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

	oldExecuteRunbookFromGithubFunction := executeRunbookFromGithubFunc
	defer func() { executeRunbookFromGithubFunc = oldExecuteRunbookFromGithubFunction }()
	executeRunbookFromGithubFunc = mockExecuteRunbookFromGithub

	closeAction := mockActionMappings["Close"]
	cmdOut, cmdErr, err := ExecuteRunbook(&closeAction, "")

	assert.NoError(t, err)
	assert.Equal(t, "", cmdOut)
	assert.Equal(t, "", cmdErr)
	assert.True(t, executeRunbookFromGithubCalled)
	executeRunbookFromGithubCalled = false
	assert.False(t, executeRunbookFromLocalCalled)
}

func TestExecuteRunbookLocal(t *testing.T) {

	oldExecuteRunbookFromLocalFunction := executeRunbookFromLocalFunc
	defer func() { executeRunbookFromLocalFunc = oldExecuteRunbookFromLocalFunction }()
	executeRunbookFromLocalFunc = mockExecuteRunbookFromLocal

	createAction := mockActionMappings["Create"]
	cmdOut, cmdErr, err := ExecuteRunbook(&createAction, "")

	assert.NoError(t, err)
	assert.Equal(t, "", cmdOut)
	assert.Equal(t, "", cmdErr)
	assert.True(t, executeRunbookFromLocalCalled)
	executeRunbookFromLocalCalled = false
	assert.False(t, executeRunbookFromGithubCalled)
}

func TestSendResultToOpsGenie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusAccepted)

		if req.Method != "POST" {
			t.Errorf("Expected ‘POST’ request, got ‘%s’", req.Method)
		}

		apiKey := req.Header.Get("Authorization")
		if apiKey != "GenieKey testKey" {
			t.Errorf("Expected request to have ‘apiKey=GenieKey testKey’, got: ‘%s’", apiKey)
		}

		contentType := req.Header.Get("Content-Type")
		if contentType != "application/json; charset=UTF-8" {
			t.Errorf("Expected request to have ‘Content-Type=application/json; charset=UTF-8’, got: ‘%s’", contentType)
		}

		actionResult := &ActionResultPayload{}
		body, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(body, actionResult)

		req.Body.Close()

		if actionResult.Action != "testAction" {
			t.Errorf("Expected request to have ‘mappedAction=testAction’, got: ‘%s’", actionResult.Action)
		}

		if actionResult.AlertId != "testAlert" {
			t.Errorf("Expected request to have ‘alertId=testAlert’, got: ‘%s’", actionResult.AlertId)
		}

		if actionResult.IsSuccessful != true {
			t.Errorf("Expected request to have ‘success=true’, got: ‘%t’", actionResult.IsSuccessful)
		}

		if actionResult.FailureMessage != "fail" {
			t.Errorf("Expected request to have ‘failureMessage=true’, got: ‘%s’", actionResult.FailureMessage)
		}

	}))
	defer ts.Close()

	actionResult := &ActionResultPayload{
		Action:"testAction",
		AlertId:"testAlert",
		IsSuccessful:true,
		FailureMessage:"fail",
	}

	apiKey := "testKey"
	err := SendResultToOpsGenie(actionResult, &apiKey, &ts.URL)

	assert.Nil(t, err)
}

func TestCannotSendResultToOpsGenie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

	}))
	defer ts.Close()

	apiKey := "testKey"
	err := SendResultToOpsGenie(new(ActionResultPayload), &apiKey, &ts.URL)

	assert.Error(t, err, "Could not send action result to OpsGenie. HttpStatus: 400")
}

func TestSendResultToOpsGenieClientError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))
	defer ts.Close()

	defer func() {client.DoFunc = nil }()
	client.DoFunc = func(retryer *retryer.Retryer, request *retryer.Request) (*http.Response, error) {
		return nil, errors.New("Test client error")
	}

	apiKey := "testKey"
	err := SendResultToOpsGenie(new(ActionResultPayload), &apiKey, &ts.URL)

	assert.Error(t, err, "Could not send action result to OpsGenie. Reason: Test client error")
}