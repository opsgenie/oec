package runbook

import (
	"github.com/opsgenie/marid2/conf"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var executeRunbookFromGithubFunction = executeRunbookFromGithub
var executeRunbookFromLocalFunction = executeRunbookFromLocal

func ExecuteRunbook(action string) (string, string, error) {
	var mappedAction = conf.RunbookActionMapping[action].(map[string]interface{})

	if len(mappedAction) > 0 {
		runbookSource := mappedAction["source"].(string)
		runbookEnvironmentVariables := mappedAction["environmentVariables"].(map[string]interface{})

		if runbookSource == "github" {
			runbookRepoOwner := mappedAction["repoOwner"].(string)
			runbookRepoName := mappedAction["repoName"].(string)
			runbookRepoFilePath := mappedAction["repoFilePath"].(string)
			runbookRepoToken := mappedAction["repoToken"].(string)

			return executeRunbookFromGithubFunction(runbookRepoOwner, runbookRepoName, runbookRepoFilePath, runbookRepoToken,
				runbookEnvironmentVariables)
		} else if runbookSource == "local" {
			runbookFilePath := mappedAction["filePath"].(string)

			return executeRunbookFromLocalFunction(runbookFilePath, runbookEnvironmentVariables)
		} else {
			return "", "", errors.New("Unknown runbook source [" + runbookSource + "].")
		}
	} else {
		return "", "", errors.New("No mapped action found for the action [" + action + "].")
	}
}

// when calling, set uri="https://api.opsgenie.com/v1/integrations/maridv2/actionExecutionResult"
func sendResultToOpsGenie(action string, alertId string, params map[string]interface{}, failureMessage string) {
	parameters := url.Values{}

	if params != nil {
		mappedAction := params["mappedActionV2"].(map[string]interface{})
		mappedActionName := mappedAction["name"].(string)
		log.Println("Sending result to OpsGenie for action: ", mappedActionName)
		parameters.Add("mappedAction", mappedActionName)

		alertId = params["alertId"].(string)
	} else {
		log.Println("Sending result to OpsGenie for action: ", action)
		parameters.Add("alertAction", action)
	}

	parameters.Add("apiKey", conf.Configuration["apiKey"].(string))

	var success bool
	if success = true; len(failureMessage) > 0 {
		success = false
		parameters.Add("failureMessage", failureMessage)
	}
	parameters.Add("success", strconv.FormatBool(success))
	parameters.Add("alertId", alertId)

	var uri strings.Builder
	uri.WriteString(conf.Configuration["opsgenieApiUrl"].(string)) // To do: finalize it before releasing
	uri.WriteString("/v1/integrations/maridv2/actionExecutionResult")

	request, err := http.NewRequest("POST", uri.String(), strings.NewReader(parameters.Encode()))
	if err != nil {
		log.Println("Could not send action result to OpsGenie. Reason: ", err)
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Could not send action result to OpsGenie. Reason: ", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("Could not read response body. Reason: ", err)
		}

		var logSuffix strings.Builder
		logSuffix.WriteString("")
		if len(string(body)) > 0 {
			logSuffix.WriteString(", Content: ")
			logSuffix.WriteString(string(body))
		}

		log.Println("Could not send action result to OpsGenie. HttpStatus: ", response.StatusCode, logSuffix)
	} else {
		log.Println("Successfully sent result to OpsGenie.")
	}
}
