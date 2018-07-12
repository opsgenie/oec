package runbook

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
)

func TestExecuteRunbookFromLocal(t *testing.T) {
	var testScriptPath = os.TempDir() + string(os.PathSeparator) + "test.sh"
	err := createTestScriptFile("echo \"Test output\"\n", testScriptPath)
	defer os.Remove(testScriptPath)

	if err != nil {
		t.Error("Error occurred while creating test file. Error: " + err.Error())
	}

	cmdOut, cmdErr, err := executeRunbookFromLocal(testScriptPath, nil)

	assert.NoError(t, err, "Error from execute operation was not empty.")
	assert.Equal(t, "Test output\n", cmdOut, "Output stream was not equal to expected.")
	assert.Equal(t, "", cmdErr, "Error stream from executed file was not empty.")
}
