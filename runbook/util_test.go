package runbook

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestWriteContentToTemporaryFile(t *testing.T) {
	var fileExt = ".txt"
	filePath, err := writeContentToTemporaryFile([]byte("test"), fileExt)
	defer os.Remove(filePath)
	assert.NoError(t, err)
	filePath2, err2 := writeContentToTemporaryFile([]byte("test 2"), fileExt)
	defer os.Remove(filePath2)
	assert.NoError(t, err2)
	assert.NotEqual(t, filePath, filePath2, "Could not create a unique file for temporary file.")
}

func TestCreateTestScriptFile(t *testing.T) {
	createTestScriptFile([]byte("test"), os.TempDir()+"test.txt")
	defer os.Remove(os.TempDir() + "test.txt")
	assert.FileExists(t, os.TempDir()+"test.txt")
}
