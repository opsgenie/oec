package conf

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
	"github.com/pkg/errors"
)

var readConfigurationFromGitCalled = false
var readConfigurationFromLocalCalled = false

var mockConf = &Configuration{
	ApiKey: 		"ApiKey",
	ActionMappings: mockActionMappings,
}

var mockActionMappings = (map[ActionName]MappedAction)(ActionMappings{
	"Create" : MappedAction{
		Source:               "local",
		FilePath:             "/path/to/runbook.bin",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
	"Close" : MappedAction{
		Source:               "github",
		RepoName:             "testRepo",
		RepoOwner:            "testAccount",
		RepoToken:            "testToken",
		RepoFilePath:         "marid/testConfig.json",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
})

var testLocalConfFilePath = "/path/to/test/conf/file.json"

func mockReadConfigurationFromGit(url string, confPath string, privateKeyFilePath string, passPhrase string) (*Configuration, error) {
	readConfigurationFromGitCalled = true

	if len(url) <= 0 {
		return nil, errors.New("URL was empty.")
	}

	if len(confPath) <= 0 {
		return nil, errors.New("confPath was empty.")
	}

	if len(privateKeyFilePath) <= 0 {
		return nil, errors.New("privateKeyFilePath was empty.")
	}

	if len(passPhrase) <= 0 {
		return nil, errors.New("passPhrase was empty.")
	}

	return mockConf, nil
}

func mockReadConfigurationFromLocalWithDefaultPath(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true
	homePath, err := getHomePath()

	if err != nil {
		return nil, err
	}

	if confPath != homePath + string(os.PathSeparator) + ".opsgenie" + string(os.PathSeparator) +
		"maridConfig.json" {
		return nil, errors.New("confPath was not as the same as the default path.")
	}

	return mockConf, nil
}

func mockReadConfigurationFromLocalWithDefaultPathWithoutActionMappings(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true
	homePath, err := getHomePath()

	if err != nil {
		return nil, err
	}

	if confPath != homePath + string(os.PathSeparator) + ".opsgenie" + string(os.PathSeparator) +
		"maridConfig.json" {
		return nil, errors.New("confPath was not as the same as the default path.")
	}

	var testConfMapWithoutActionMappings = &Configuration{}

	return testConfMapWithoutActionMappings, nil
}

func mockReadConfigurationFromLocalWithCustomPath(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true

	if confPath != testLocalConfFilePath {
		return nil, errors.New("confPath was not as the same as the default path.")
	}

	return mockConf, nil
}

func TestReadConfFileFromGit(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "git")
	os.Setenv("MARIDCONFREPOPRIVATEKEYPATH", "/path/to/private/key.pem")
	os.Setenv("MARIDCONFREPOGITURL", "https://github.com/acc/repo.git")
	os.Setenv("MARIDCONFGITFILEPATH", "marid/testConf.json")
	os.Setenv("MARIDCONFGITPASSPHRASE", "passPhrase")

	oldReadFromGitFunction := readConfigurationFromGitFunction
	defer func() {readConfigurationFromGitFunction = oldReadFromGitFunction }()
	readConfigurationFromGitFunction = mockReadConfigurationFromGit

	configuration, err := ReadConfFile()

	if err != nil {
		t.Error("Error occurred while calling ReadConfFile. Error: " + err.Error())
	}

	assert.True(t, readConfigurationFromGitCalled,
		"ReadConfFile did not call the method readConfigurationFromGit.")
	readConfigurationFromGitCalled = false
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")

	assert.Equal(t, mockConf, configuration,
		"Global configuration was not equal to given configuration.")

}

func TestReadConfFileFromLocalWithDefaultPath(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")

	oldReadFromLocalFunction := readConfigurationFromLocalFunction
	defer func() { readConfigurationFromLocalFunction = oldReadFromLocalFunction }()
	readConfigurationFromLocalFunction = mockReadConfigurationFromLocalWithDefaultPath

	configuration, err := ReadConfFile()

	if err != nil {
		t.Error("Error occurred while calling ReadConfFile. Error: " + err.Error())
	}

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.Equal(t, mockConf, configuration,
		"Global configuration was not equal to given configuration.")
}

func TestReturnErrorIfActionMappingsNotFoundInTheConfFile(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")

	oldReadFromLocalFunction := readConfigurationFromLocalFunction
	defer func() { readConfigurationFromLocalFunction = oldReadFromLocalFunction }()
	readConfigurationFromLocalFunction = mockReadConfigurationFromLocalWithDefaultPathWithoutActionMappings
	_, err := ReadConfFile()

	assert.Error(t, err, "Error should be thrown because action mappings do not exist in the configuration.")
	assert.Equal(t, "Action mappings configuration is not found in the configuration file.", err.Error(),
		"Error message was not equal to expected.")

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
}

func TestReadConfFileFromLocalWithCustomPath(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")
	os.Setenv("MARIDCONFLOCALFILEPATH", testLocalConfFilePath)

	oldReadFromLocalFunction := readConfigurationFromLocalFunction
	defer func() {readConfigurationFromLocalFunction = oldReadFromLocalFunction}()
	readConfigurationFromLocalFunction = mockReadConfigurationFromLocalWithCustomPath
	configuration, err := ReadConfFile()

	if err != nil {
		t.Error("Error occurred while calling ReadConfFile. Error: " + err.Error())
	}

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.Equal(t, configuration, mockConf,
		"Global configuration was not equal to given configuration.")
}

func TestReadConfFileWithUnknownSource(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "sacma")

	_, err := ReadConfFile()
	assert.Error(t, err, "Error should be thrown.")

	if err.Error() != "Unknown configuration source [sacma]." {
		t.Error("Error message was wrong.")
	}

	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")
}
