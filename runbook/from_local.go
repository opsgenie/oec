package runbook

func executeRunbookFromLocal(executablePath string, environmentVariables map[string]interface{}) (string, string, error) {
	return execute(executablePath, nil, environmentVariables)
}
