package util

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCreateTempTestScriptFile(t *testing.T) {
	var fileExt = ".txt"
	filePath, err := CreateTempTestFile([]byte("test"), fileExt)
	assert.FileExists(t, filePath)
	assert.NoError(t, err)
	os.Remove(filePath)

	filePath2, err2 := CreateTempTestFile([]byte("test 2"), fileExt)
	assert.FileExists(t, filePath2)
	assert.NoError(t, err2)
	os.Remove(filePath2)

	assert.NotEqual(t, filePath, filePath2, "Could not create a unique file for temporary file.")
}
