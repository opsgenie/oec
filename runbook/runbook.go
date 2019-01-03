package runbook

import (
	"bytes"
	"encoding/json"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/retryer"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strconv"
)

var ResultPath = "/v2/integrations/maridv2/actionExecutionResult"

var executeRunbookFromGithubFunc = executeRunbookFromGithub
var executeRunbookFromLocalFunc = executeRunbookFromLocal
var ExecuteRunbookFunc = ExecuteRunbook
var SendResultToOpsGenieFunc = SendResultToOpsGenie

var client = &retryer.Retryer{}

func ExecuteRunbook(mappedAction *conf.MappedAction, arg string) (string, string, error) {

	source := mappedAction.Source
	environmentVariables := mappedAction.EnvironmentVariables

	switch source {
	case "github":
		repoOwner := mappedAction.RepoOwner
		repoName := mappedAction.RepoName
		repoFilePath := mappedAction.RepoFilePath
		repoToken := mappedAction.RepoToken

		return executeRunbookFromGithubFunc(repoOwner, repoName, repoFilePath, repoToken, []string{arg}, environmentVariables)
	case "local":
		filePath := mappedAction.FilePath

		return executeRunbookFromLocalFunc(filePath, []string{arg}, environmentVariables)
	default:
		return "", "", errors.Errorf("Unknown runbook source [%s].", source)
	}
}

func SendResultToOpsGenie(resultPayload *ActionResultPayload, apiKey, baseUrl *string) error {

	body, err := json.Marshal(resultPayload)
	if err != nil {
		return  errors.Errorf("Cannot marshall payload: %s", err)
	}

	resultUrl := *baseUrl + ResultPath

	request, err := retryer.NewRequest("POST", resultUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", "GenieKey " + *apiKey)
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {

		errorMessage := "Unexpected response status: " + strconv.Itoa(response.StatusCode)

		body, err := ioutil.ReadAll(response.Body)
		if err == nil {
			return errors.Errorf(errorMessage + ", error message: %s", string(body))
		} else {
			return errors.Errorf(errorMessage + ", also could not read response body: %s", err)
		}
	}

	return nil
}