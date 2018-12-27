package runbook

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var executables = map[string]string{
	".bat" 	: "cmd",
	".cmd" 	: "cmd",
	".ps1" 	: "powershell",
	".sh"	: "sh",
}

func execute(executablePath string, args []string, environmentVariables []string) (string, string, error) {

	fileExt := filepath.Ext(strings.ToLower(executablePath))
	executable, _ := executables[fileExt]

	var cmd *exec.Cmd

	if executable == "cmd" {
		cmd = exec.Command(executable, append([]string{"/C", executablePath}, args...)...)
	} else if executable == "sh" || executable == "powershell" {
		cmd = exec.Command(executable, append([]string{executablePath}, args...)...)
	} else {
		cmd = exec.Command(executablePath, args...)
	}

	var cmdOutput, cmdErr bytes.Buffer

	cmd.Stdout = &cmdOutput
	cmd.Stderr = &cmdErr
	cmd.Env = append(os.Environ(), environmentVariables...)

	err := cmd.Run()
	if err != nil {
		return "", "", err
	}

	return cmdOutput.String(), cmdErr.String(), nil
}
