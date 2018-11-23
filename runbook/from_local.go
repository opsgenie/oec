package runbook

func executeRunbookFromLocal(executablePath string, args []string, environmentVariables map[string]interface{}) (string, string, error) {
	return execute(executablePath, args, environmentVariables)
}
