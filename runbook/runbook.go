package runbook

import (
	"bytes"
	"encoding/json"
	"github.com/opsgenie/ois/conf"
	"github.com/opsgenie/ois/git"
	"github.com/opsgenie/ois/retryer"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	fpath "path/filepath"
	"strconv"
)

var resultPath = "/v2/integrations/ois/actionExecutionResult"

var ExecuteRunbookFunc = ExecuteRunbook
var SendResultToOpsGenieFunc = SendResultToOpsGenie

var client = &retryer.Retryer{}

func ExecuteRunbook(mappedAction *conf.MappedAction, repositories *git.Repositories, args []string) (string, string, error) {

	source := mappedAction.SourceType
	environmentVariables := mappedAction.EnvironmentVariables
	filepath := mappedAction.Filepath

	switch source {
	case conf.LocalSourceType:
		return executeFunc(filepath, args, environmentVariables)

	case conf.GitSourceType:
		if repositories == nil {
			return "", "", errors.New("Repositories should be provided.")
		}

		url := mappedAction.GitOptions.Url

		repository, err := repositories.Get(url)
		if err != nil {
			return "", "", err
		}

		repository.RLock()
		defer repository.RUnlock()

		filepath = fpath.Join(repository.Path, filepath)

		return executeFunc(filepath, args, environmentVariables)

	default:
		return "", "", errors.Errorf("Unknown runbook source [%s].", source)
	}
}

func SendResultToOpsGenie(resultPayload *ActionResultPayload, apiKey, baseUrl *string) error {

	body, err := json.Marshal(resultPayload)
	if err != nil {
		return errors.Errorf("Cannot marshall payload: %s", err)
	}

	resultUrl := *baseUrl + resultPath

	request, err := retryer.NewRequest("POST", resultUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", "GenieKey "+*apiKey)
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
			return errors.Errorf(errorMessage+", error message: %s", string(body))
		} else {
			return errors.Errorf(errorMessage+", also could not read response body: %s", err)
		}
	}

	return nil
}
