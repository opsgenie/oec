package runbook

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var executeFunc = execute

var executables = map[string]string{
	".bat" 	: "cmd",
	".cmd" 	: "cmd",
	".ps1" 	: "powershell",
	".sh"	: "sh",
}

func execute(executablePath string, args []string, environmentVars []string) (string, string, error) {

	fileExt := filepath.Ext(strings.ToLower(executablePath))
	executable, _ := executables[fileExt]

	if args == nil {
		args = []string{}
	} else if environmentVars == nil {
		environmentVars = []string{}
	}

	var cmd *exec.Cmd

	switch executable {
	case "cmd":
		cmd = exec.Command(executable, append([]string{"/C", executablePath}, args...)...)
	case "sh", "powershell":
		cmd = exec.Command(executable, append([]string{executablePath}, args...)...)
	default:
		cmd = exec.Command(executablePath, args...)
	}

	var cmdOutput, cmdErr bytes.Buffer

	cmd.Stdout = &cmdOutput
	cmd.Stderr = &cmdErr
	cmd.Env = append(os.Environ(), environmentVars...)

	err := cmd.Run()
	if err != nil {
		return "", "", err
	}

	return cmdOutput.String(), cmdErr.String(), nil
}
