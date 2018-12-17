package runbook

import (
	"bytes"
	"encoding/json"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/retryer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
)

//const resultUrl = "https://api.opsgenie.com/v1/integrations/maridv2/actionExecutionResult"
var resultPath = "/v2/integrations/maridv2/actionExecutionResult" // todo read conf base url

var executeRunbookFromGithubFunc = executeRunbookFromGithub
var executeRunbookFromLocalFunc = executeRunbookFromLocal
var ExecuteRunbookFunc = ExecuteRunbook

var client = &retryer.Retryer{}

func ExecuteRunbook(mappedAction *conf.MappedAction, arg string) (string, string, error) {

	source := mappedAction.Source
	environmentVariables := mappedAction.EnvironmentVariables

	if source == "github" {
		repoOwner := mappedAction.RepoOwner
		repoName := mappedAction.RepoName
		repoFilePath := mappedAction.RepoFilePath
		repoToken := mappedAction.RepoToken

		return executeRunbookFromGithubFunc(repoOwner, repoName, repoFilePath, repoToken, []string{arg}, environmentVariables)
	} else if source == "local" {
		runbookFilePath := mappedAction.FilePath

		return executeRunbookFromLocalFunc(runbookFilePath, []string{arg}, environmentVariables)
	} else {
		return "", "", errors.New("Unknown runbook source [" + source + "].")
	}
}

func SendResultToOpsGenie(resultPayload *ActionResultPayload, apiKey *string, baseUrl *string) {

	body, err := json.Marshal(resultPayload)
	if err != nil {
		logrus.Error("Cannot marshall payload: ", err)
		return
	}

	resultUrl := *baseUrl + resultPath

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

	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {

		logMessage := "Could not send action result to OpsGenie. HttpStatus: " + strconv.Itoa(response.StatusCode)

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			logrus.Error(logMessage, ". Could not read response body. Reason: ", err)
		} else {
			logrus.Error(logMessage, ". Error message: " , body)
		}
	} else {
		logrus.Debug("Successfully sent result to OpsGenie.")
	}
}
