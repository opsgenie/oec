package runbook

import (
	"bytes"
	"os/exec"
	"os"
	"strings"
)

func execute(executablePath string, args []string, environmentVariables map[string]interface{}) (string, string, error) {
	executable := determineExecutable(executablePath)
	var cmd *exec.Cmd

	if executable == "cmd" {
		cmd = exec.Command(executable, append([]string{"/C", executablePath}, args...)...)
	} else if executable == "sh" || executable == "powershell" {
		cmd = exec.Command(executable, append([]string{executablePath}, args...)...)
	} else {
		cmd = exec.Command(executablePath, args...)
	}

	cmdOutput := &bytes.Buffer{}
	cmdErr := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	cmd.Stderr = cmdErr
	env := os.Environ()
	env = append(env, convertMapToArray(environmentVariables)...)
	cmd.Env = env
	err := cmd.Run()
	commandOutput := cmdOutput.String()
	errorOutput := cmdErr.String()

	if err != nil {
		return "", "", err
	}

	return commandOutput, errorOutput, nil
}

func determineExecutable(executablePath string) string {
	filePathInLowerCase := strings.ToLower(executablePath)

	if strings.HasSuffix(filePathInLowerCase, ".bat") ||
		strings.HasSuffix(filePathInLowerCase, ".cmd") {
		return "cmd"
	} else if strings.HasSuffix(filePathInLowerCase, ".ps1") {
		return "powershell"
	} else if strings.HasSuffix(filePathInLowerCase, ".sh") {
		return "sh"
	} else {
		return ""
	}
}
