package conf

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
)

var readConfigurationFromGitCalled = false
var readConfigurationFromLocalCalled = false
var testConfMap = map[string]string{
	"key1": "val1",
	"key2": "val2",
}

func mockReadConfigurationFromGit(url string, username string, password string) (map[string]string, error) {
	readConfigurationFromGitCalled = true

	return testConfMap, nil
}

func mockReadConfigurationFromLocal() (map[string]string, error) {
	readConfigurationFromLocalCalled = true

	return testConfMap, nil
}

func TestReadConfFileFromGit(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "git")
	os.Setenv("MARIDCONFREPOUSERNAME", "testUserName")
	os.Setenv("MARIDCONFREPOPASSWORD", "testPassword")
	os.Setenv("MARIDCONFREPOGITURL", "https://github.com/acc/repo.git")

	oldReadFromGitFunction := readConfigurationFromGitFunction
	defer func() {readConfigurationFromGitFunction = oldReadFromGitFunction }()
	readConfigurationFromGitFunction = mockReadConfigurationFromGit
	err := ReadConfFile()

	if err != nil {
		t.Error("Error occurred while calling ReadConfFile. Error: " + err.Error())
	}

	assert.True(t, readConfigurationFromGitCalled,
		"ReadConfFile did not call the method readConfigurationFromGit.")
	readConfigurationFromGitCalled = false
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")
	assert.Equal(t, Configuration, testConfMap,
		"Global configuration was not equal to given configuration.")
}

func TestReadConfFileFromLocal(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")

	oldReadFromLocalFunction := readConfigurationFromLocalFunction
	defer func() {readConfigurationFromLocalFunction = oldReadFromLocalFunction}()
	readConfigurationFromLocalFunction = mockReadConfigurationFromLocal
	err := ReadConfFile()

	if err != nil {
		t.Error("Error occurred while calling ReadConfFile. Error: " + err.Error())
	}

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.Equal(t, Configuration, testConfMap,
		"Global configuration was not equal to given configuration.")
}

func TestReadConfFileWithUnknownSource(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "sacma")

	err := ReadConfFile()
	assert.Error(t, err, "Error should be thrown.")

	if err.Error() != "Unknown configuration source [sacma]." {
		t.Error("Error message was wrong.")
	}

	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")
}