package runbook

import (
	"encoding/json"
	"github.com/opsgenie/oec/retryer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

		if actionResult.EntityId != "testAlert" {
			t.Errorf("Expected request to have ‘entityId=testAlert’, got: ‘%s’", actionResult.EntityId)
		}

		if actionResult.IsSuccessful != true {
			t.Errorf("Expected request to have ‘success=true’, got: ‘%t’", actionResult.IsSuccessful)
		}

		if actionResult.FailureMessage != "fail" {
			t.Errorf("Expected request to have ‘resultMessage=true’, got: ‘%s’", actionResult.FailureMessage)
		}

	}))
	defer ts.Close()

	actionResult := &ActionResultPayload{
		Action:         "testAction",
		EntityId:       "testAlert",
		IsSuccessful:   true,
		FailureMessage: "fail",
	}

	apiKey := "testKey"
	err := SendResultToOpsGenie(actionResult, apiKey, ts.URL)

	assert.Nil(t, err)
}

func TestCannotSendResultToOpsGenie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

	}))
	defer ts.Close()

	apiKey := "testKey"
	err := SendResultToOpsGenie(new(ActionResultPayload), apiKey, ts.URL)

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
	err := SendResultToOpsGenie(new(ActionResultPayload), apiKey, ts.URL)

	assert.Error(t, err, "Could not send action result to OpsGenie. Reason: Test client error")
}
