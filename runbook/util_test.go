package runbook

import (
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func TestWriteContentToTemporaryFile(t *testing.T) {
	var fileName = "test.txt"
	filePath, err := writeContentToTemporaryFile([]byte("test"), fileName)
	defer os.Remove(filePath)
	assert.NoError(t, err)
	filePath2, err2 := writeContentToTemporaryFile([]byte("test 2"), fileName)
	defer os.Remove(filePath2)
	assert.NoError(t, err2)
	assert.NotEqual(t, filePath, filePath2, "Could not create a unique file for temporary file.")
}

func TestAppendUniqueRandomPostfixToFileName(t *testing.T) {
	fileName := "test.sh"
	result, err := appendUniqueRandomPostfixToFileName(fileName)
	assert.NoError(t, err)
	assert.NotEqual(t, fileName, result)
	assert.True(t, strings.HasPrefix(result, "test"))
	assert.True(t, strings.HasSuffix(result, ".sh"))
	fileName = "test"
	result, err = appendUniqueRandomPostfixToFileName(fileName)
	assert.NoError(t, err)
	assert.NotEqual(t, fileName, result)
	assert.True(t, strings.HasPrefix(result, "test"))
	assert.False(t, strings.HasSuffix(result, ".sh"))
}

func TestCreateTestScriptFile(t *testing.T) {
	createTestScriptFile("test", os.TempDir()+"test.txt")
	defer os.Remove(os.TempDir() + "test.txt")
	assert.FileExists(t, os.TempDir()+"test.txt")
}
