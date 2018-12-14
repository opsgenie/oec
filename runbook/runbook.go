package runbook

import (
	"github.com/opsgenie/marid2/conf"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"github.com/sirupsen/logrus"
	"encoding/json"
	"bytes"
	"github.com/opsgenie/marid2/retryer"
)

//const resultUrl = "https://api.opsgenie.com/v1/integrations/maridv2/actionExecutionResult"
var resultUrl = "https://a2e3dfe8.ngrok.io/v2/integrations/maridv2/actionExecutionResult"

var executeRunbookFromGithubFunc = executeRunbookFromGithub
var executeRunbookFromLocalFunc = executeRunbookFromLocal
var ExecuteRunbookFunc = ExecuteRunbook

var client = &retryer.Retryer{}

func ExecuteRunbook(mappedAction *conf.MappedAction, arg string) (string, string, error) {

	runbookSource := mappedAction.Source
	runbookEnvironmentVariables := mappedAction.EnvironmentVariables

	if runbookSource == "github" {
		repoOwner := mappedAction.RepoOwner
		repoName := mappedAction.RepoName
		repoFilePath := mappedAction.RepoFilePath
		repoToken := mappedAction.RepoToken

		return executeRunbookFromGithubFunc(repoOwner, repoName, repoFilePath, repoToken, []string{arg}, runbookEnvironmentVariables)
	} else if runbookSource == "local" {
		runbookFilePath := mappedAction.FilePath

		return executeRunbookFromLocalFunc(runbookFilePath, []string{arg}, runbookEnvironmentVariables)
	} else {
		return "", "", errors.New("Unknown runbook source [" + runbookSource + "].")
	}
}

func SendResultToOpsGenie(resultPayload *ActionResultPayload, apiKey *string) {

	body, err := json.Marshal(resultPayload)
	if err != nil {
		logrus.Error("Cannot marshall payload: ", err)
		return
	}

	request, err := http.NewRequest("POST", resultUrl, bytes.NewBuffer(body))
	if err != nil {
		logrus.Error("Could not send action result to OpsGenie. Reason: ", err)
		return
	}
	request.Header.Add("Authorization", "GenieKey " + *apiKey)
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")

	response, err := client.Do(request)
	if err != nil {
		logrus.Error("Could not send action result to OpsGenie. Reason: ", err)
		return
	}

	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			logrus.Error("Could not read response body. Reason: ", err)
		}

		logrus.Error("Could not send action result to OpsGenie. HttpStatus: ", response.StatusCode, ", Error message:" , string(body))
	} else {
		logrus.Debug("Successfully sent result to OpsGenie.")
	}
}
