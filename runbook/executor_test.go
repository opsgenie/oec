package runbook

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/opsgenie/oec/util"
	"github.com/stretchr/testify/assert"
)

const shFileExt = ".sh"
const batFileExt = ".bat"

func TestExecuteSuccess(t *testing.T) {
	testEnvironmentVariables := []string{"TESTENVVAR=test env var", "ANOTHERVAR=another"}

	if runtime.GOOS != "windows" {
		content := []byte("echo \"Test output\"\necho \"Given Environment Variable: TESTENVVAR: $TESTENVVAR\"\n")
		tmpFilePath, err := util.CreateTempTestFile(content, shFileExt)
		defer os.Remove(tmpFilePath)

		if err != nil {
			t.Error(err.Error())
		}

		cmdOutput, cmdErr := &bytes.Buffer{}, &bytes.Buffer{}
		err = Execute(tmpFilePath, nil, testEnvironmentVariables, cmdOutput, cmdErr)

		assert.NoError(t, err, "Error from Execute operation was not empty.")
		assert.Equal(t, "", cmdErr.String(), "Error stream from executed file was not empty.")
		assert.Equal(t, "Test output\nGiven Environment Variable: TESTENVVAR: test env var\n", cmdOutput.String(),
			"Output stream was not equal to expected.")
	} else {
		content := []byte("@echo off\r\necho Test output\r\necho Given Environment Variable: TESTENVVAR: %TESTENVVAR%\n")
		tmpFilePath, err := util.CreateTempTestFile(content, batFileExt)
		defer os.Remove(tmpFilePath)

		if err != nil {
			t.Error(err.Error())
		}

		cmdOutput, cmdErr := &bytes.Buffer{}, &bytes.Buffer{}
		err = Execute(tmpFilePath, nil, testEnvironmentVariables, cmdOutput, cmdErr)

		assert.NoError(t, err, "Error from Execute operation was not empty.")
		assert.Equal(t, "", cmdErr.String(), "Error stream from executed file was not empty.")
		assert.Equal(t, "Test output\r\nGiven Environment Variable: TESTENVVAR: test env var\r\n", cmdOutput.String(),
			"Output stream was not equal to expected.")
	}
}

func TestExecuteWithErrorStream(t *testing.T) {
	if runtime.GOOS != "windows" {
		content := []byte(">&2 echo \"test error\"\n")
		tmpFilePath, err := util.CreateTempTestFile(content, shFileExt)
		defer os.Remove(tmpFilePath)

		if err != nil {
			t.Error(err.Error())
		}

		cmdOutput, cmdErr := &bytes.Buffer{}, &bytes.Buffer{}
		err = Execute(tmpFilePath, nil, nil, cmdOutput, cmdErr)

		assert.NoError(t, err, "Error from Execute operation was not empty.")
		assert.Equal(t, "", cmdOutput.String(), "Output stream from executed file was not empty.")
		assert.Equal(t, "test error\n", cmdErr.String(), "Error stream was not equal to expected.")
	} else {
		content := []byte("@echo off\r\necho test error>&2\r\n")
		tmpFilePath, err := util.CreateTempTestFile(content, batFileExt)
		defer os.Remove(tmpFilePath)

		if err != nil {
			t.Error(err.Error())
		}

		cmdOutput, cmdErr := &bytes.Buffer{}, &bytes.Buffer{}
		err = Execute(tmpFilePath, nil, nil, cmdOutput, cmdErr)

		assert.NoError(t, err, "Error from Execute operation was not empty.")
		assert.Equal(t, "", cmdOutput.String(), "Output stream from executed file was not empty.")
		assert.Equal(t, "test error\r\n", cmdErr.String(), "Error stream was not equal to expected.")
	}
}

func TestExecuteWithError(t *testing.T) {
	switch goos := runtime.GOOS; goos {
	case "darwin":
		content := []byte("sacmasapan")
		tmpFilePath, err := util.CreateTempTestFile(content, shFileExt)
		defer os.Remove(tmpFilePath)

		if err != nil {
			t.Error(err.Error())
		}

		cmdOutput, cmdErr := &bytes.Buffer{}, &bytes.Buffer{}
		err = Execute(tmpFilePath, nil, nil, cmdOutput, cmdErr)

		assert.IsType(t, &ExecError{}, err)
		assert.Error(t, err, "Error from Execute operation was empty.")
		assert.Equal(t, err.Error(), "exit status 127", "Error message was not equal to expected.")
		assert.Equal(t, "", cmdOutput.String(), "Output stream from executed file was not empty.")
		assert.Contains(t, cmdErr.String(), "command not found", "Error stream from executed file does not contain err message.")
		assert.Contains(t, err.(*ExecError).Stderr, cmdErr.String(), "ExecError is not same as cmdErr.")
	case "windows":
		content := []byte("sacmasapan")
		tmpFilePath, err := util.CreateTempTestFile(content, batFileExt)
		defer os.Remove(tmpFilePath)

		if err != nil {
			t.Error(err.Error())
		}

		cmdOutput, cmdErr := &bytes.Buffer{}, &bytes.Buffer{}
		err = Execute(tmpFilePath, nil, nil, cmdOutput, cmdErr)

		assert.IsType(t, &ExecError{}, err)
		assert.Error(t, err, "Error from Execute operation was empty.")
		assert.Equal(t, err.Error(), "exit status 1", "Error message was not equal to expected.")
		assert.Contains(t, cmdErr.String(), "not recognized as an internal or external command",
			"Error stream from executed file does not contain err message.")
		assert.Contains(t, err.(*ExecError).Stderr, cmdErr.String(), "ExecError is not same as cmdErr.")
	case "linux":
		content := []byte("sacmasapan")
		tmpFilePath, err := util.CreateTempTestFile(content, shFileExt)
		defer os.Remove(tmpFilePath)

		if err != nil {
			t.Error(err.Error())
		}

		cmdOutput, cmdErr := &bytes.Buffer{}, &bytes.Buffer{}
		err = Execute(tmpFilePath, nil, nil, cmdOutput, cmdErr)

		assert.IsType(t, &ExecError{}, err)
		assert.Error(t, err, "Error from Execute operation was empty.")
		assert.Equal(t, err.Error(), "exit status 127", "Error message was not equal to expected.")
		assert.Equal(t, "", cmdOutput.String(), "Output stream from executed file was not empty.")
		assert.Contains(t, cmdErr.String(), "not found", "Error stream from executed file does not contain err message.")
		assert.Contains(t, err.(*ExecError).Stderr, cmdErr.String(), "ExecError is not same as cmdErr.")
	}
}
