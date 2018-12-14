package runbook

func executeRunbookFromLocal(executablePath string, args []string, environmentVariables []string) (string, string, error) {
	return execute(executablePath, args, environmentVariables)
}
