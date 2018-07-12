package conf

import (
	"testing"
	"os"
	"github.com/stretchr/testify/assert"
	"github.com/pkg/errors"
)

var readConfigurationFromGitCalled = false
var readConfigurationFromLocalCalled = false
var testConfMap = map[string]interface{}{
	"actionMappings": map[string]interface{}{
		"Create": map[string]interface{}{
			"filePath": "/path/to/runbook.bin",
			"source":   "local",
			"environmentVariables": map[string]interface{}{
				"k1": "v1",
				"k2": "v2",
			},
		},
		"Close": map[string]interface{}{
			"source":       "github",
			"repoOwner":    "testAccount",
			"repoName":     "testRepo",
			"repoFilePath": "marid/testConfig.json",
			"repoToken":    "testtoken",
			"environmentVariables": map[string]interface{}{
				"k1": "v1",
				"k2": "v2",
			},
		},
	},
	"key1": "val1",
	"key2": "val2",
}
var testActionMappings = map[string]interface{}{
	"Create": map[string]interface{}{
		"filePath": "/path/to/runbook.bin",
		"source":   "local",
		"environmentVariables": map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
		},
	},
	"Close": map[string]interface{}{
		"source":       "github",
		"repoOwner":    "testAccount",
		"repoName":     "testRepo",
		"repoFilePath": "marid/testConfig.json",
		"repoToken":    "testtoken",
		"environmentVariables": map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
		},
	},
}
var testLocalConfFilePath = "/path/to/test/conf/file.json"

func mockReadConfigurationFromGit(url string, confPath string, privateKeyFilePath string) (map[string]interface{}, error) {
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

	return testConfMap, nil
}

func mockReadConfigurationFromLocalWithDefaultPath(confPath string) (map[string]interface{}, error) {
	readConfigurationFromLocalCalled = true
	homePath, err := getHomePath()

	if err != nil {
		return nil, err
	}

	if confPath != homePath+string(os.PathSeparator)+".opsgenie"+string(os.PathSeparator) +
		"maridConfig.json" {
		return nil, errors.New("confPath was not as the same as the default path.")
	}

	return testConfMap, nil
}

func mockReadConfigurationFromLocalWithDefaultPathWithoutActionMappings(confPath string) (map[string]interface{}, error) {
	readConfigurationFromLocalCalled = true
	homePath, err := getHomePath()

	if err != nil {
		return nil, err
	}

	if confPath != homePath+string(os.PathSeparator)+".opsgenie"+string(os.PathSeparator) +
		"maridConfig.json" {
		return nil, errors.New("confPath was not as the same as the default path.")
	}

	var testConfMapWithoutActionMappings = map[string]interface{}{
		"key1": "val1",
		"key2": "val2",
	}

	return testConfMapWithoutActionMappings, nil
}

func mockReadConfigurationFromLocalWithCustomPath(confPath string) (map[string]interface{}, error) {
	readConfigurationFromLocalCalled = true

	if confPath != testLocalConfFilePath {
		return nil, errors.New("confPath was not as the same as the default path.")
	}

	return testConfMap, nil
}

func TestReadConfFileFromGit(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "git")
	os.Setenv("MARIDCONFREPOPRIVATEKEYPATH", "/path/to/private/key.pem")
	os.Setenv("MARIDCONFREPOGITURL", "https://github.com/acc/repo.git")
	os.Setenv("MARIDCONFGITFILEPATH", "marid/testConf.json")

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

	assert.Equal(t, testConfMap, Configuration,
		"Global configuration was not equal to given configuration.")
	assert.Equal(t, testActionMappings, RunbookActionMapping,
		"Global action mapping was not equal to given action mapping.")
}

func TestReadConfFileFromLocalWithDefaultPath(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")

	oldReadFromLocalFunction := readConfigurationFromLocalFunction
	defer func() { readConfigurationFromLocalFunction = oldReadFromLocalFunction }()
	readConfigurationFromLocalFunction = mockReadConfigurationFromLocalWithDefaultPath
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
	assert.Equal(t, testActionMappings, RunbookActionMapping,
		"Global action mapping was not equal to given action mapping.")
}

func TestReturnErrorIfActionMappingsNotFoundInTheConfFile(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")

	oldReadFromLocalFunction := readConfigurationFromLocalFunction
	defer func() { readConfigurationFromLocalFunction = oldReadFromLocalFunction }()
	readConfigurationFromLocalFunction = mockReadConfigurationFromLocalWithDefaultPathWithoutActionMappings
	err := ReadConfFile()

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
	assert.Equal(t, testActionMappings, RunbookActionMapping,
		"Global action mapping was not equal to given action mapping.")
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
