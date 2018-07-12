package runbook

import (
	"github.com/opsgenie/marid2/conf"
	"github.com/pkg/errors"
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
