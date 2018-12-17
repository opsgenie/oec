package runbook

import (
	"bytes"
	"encoding/json"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/retryer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func mockExecuteRunbookFromGithub(runbookRepoOwner string, runbookRepoName string, runbookRepoFilePath string, runbookRepoToken string, args []string, environmentVariables []string) (string, string, error) {
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

func mockExecuteRunbookFromLocal(executablePath string, args []string, environmentVariables []string) (string, string, error) {
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)

		if r.Method != "POST" {
			t.Errorf("Expected ‘POST’ request, got ‘%s’", r.Method)
		}

		apiKey := r.Header.Get("Authorization")
		if apiKey != "GenieKey testKey" {
			t.Errorf("Expected request to have ‘apiKey=GenieKey testKey’, got: ‘%s’", apiKey)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json; charset=UTF-8" {
			t.Errorf("Expected request to have ‘Content-Type=application/json; charset=UTF-8’, got: ‘%s’", contentType)
		}

		actionResult := &ActionResultPayload{}
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, actionResult)

		r.Body.Close()

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

	logrus.SetLevel(logrus.DebugLevel)
	buffer := &bytes.Buffer{}
	logrus.SetOutput(buffer)

	apiKey := "testKey"
	SendResultToOpsGenie(actionResult, &apiKey, &ts.URL)

	assert.Contains(t, buffer.String(), "Successfully sent result to OpsGenie.")
}

func TestCannotSendResultToOpsGenie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

	}))
	defer ts.Close()

	buffer := &bytes.Buffer{}
	logrus.SetOutput(buffer)

	apiKey := "testKey"
	SendResultToOpsGenie(new(ActionResultPayload), &apiKey, &ts.URL)

	assert.Contains(t, buffer.String(), "Could not send action result to OpsGenie. HttpStatus: 400")
}

func TestSendResultToOpsGenieClientError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))
	defer ts.Close()

	defer func() {client.DoFunc = nil }()
	client.DoFunc = func(retryer *retryer.Retryer, request *http.Request) (*http.Response, error) {
		return nil, errors.New("Test client error")
	}

	buffer := &bytes.Buffer{}
	logrus.SetOutput(buffer)

	apiKey := "testKey"
	SendResultToOpsGenie(new(ActionResultPayload), &apiKey, &ts.URL)

	assert.Contains(t, buffer.String(), "Could not send action result to OpsGenie. Reason: Test client error")
}

func TestExternalSendResultToOpsGenie(t *testing.T) {

	actionResult := &ActionResultPayload{
		Action:"testAction",
		AlertId:"5c354eaa-f92f-44a5-a868-dd8fbab40dd8-1544769998862",
		IsSuccessful:true,
		FailureMessage:"fail",
	}

	apiKey := "8a22a64d-11a5-44e9-9c31-2c7bb3518458"
	baseUrl := "https://api.opsgenie.com"
	SendResultToOpsGenie(actionResult, &apiKey, &baseUrl)
}
