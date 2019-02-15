package runbook

import (
	"encoding/json"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/git"
	"github.com/opsgenie/marid2/retryer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockActionMappings = (map[conf.ActionName]conf.MappedAction)(conf.ActionMappings{
	"Create": conf.MappedAction{
		SourceType:           "local",
		Filepath:             "/path/to/runbook.bin",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
	"Close": conf.MappedAction{
		SourceType: "git",
		GitOptions: git.GitOptions{
			Url:                "testUrl",
			PrivateKeyFilepath: "testKeyPath",
			Passphrase:         "testPass",
		},
		Filepath:             "ois/testConfig.json",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
})

func TestExecuteRunbookGitNilRepositories(t *testing.T) {

	closeAction := mockActionMappings["Close"]
	_, _, err := ExecuteRunbook(&closeAction, nil, nil)

	assert.NotNil(t, err)
	assert.EqualError(t, err, "Repositories should be provided.")
}

func TestExecuteRunbookGitNonExistingRepository(t *testing.T) {

	repositories := &git.Repositories{}

	closeAction := mockActionMappings["Close"]
	_, _, err := ExecuteRunbook(&closeAction, repositories, nil)

	assert.NotNil(t, err)
	assert.EqualError(t, err, "Git repository[testUrl] could not be found.")
}

func testExecuteRunbookLocal(t *testing.T) {

	createAction := mockActionMappings["Create"]
	cmdOut, cmdErr, err := ExecuteRunbook(&createAction, nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, "", cmdOut)
	assert.Equal(t, "", cmdErr)

}

func TestExecuteRunbook(t *testing.T) {

	executeFunc = func(executablePath string, args []string, environmentVariables []string) (s string, s2 string, e error) {
		return "", "", nil
	}

	t.Run("TestExecuteRunbookLocal", testExecuteRunbookLocal)

	executeFunc = execute
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
		Action:         "testAction",
		AlertId:        "testAlert",
		IsSuccessful:   true,
		FailureMessage: "fail",
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

	defer func() { client.DoFunc = nil }()
	client.DoFunc = func(retryer *retryer.Retryer, request *retryer.Request) (*http.Response, error) {
		return nil, errors.New("Test client error")
	}

	apiKey := "testKey"
	err := SendResultToOpsGenie(new(ActionResultPayload), &apiKey, &ts.URL)

	assert.Error(t, err, "Could not send action result to OpsGenie. Reason: Test client error")
}
