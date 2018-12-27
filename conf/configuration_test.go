package conf

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var readConfigurationFromGitHubCalled = false
var readConfigurationFromLocalCalled = false

var mockConf = &Configuration{
	ApiKey: 		"ApiKey",
	BaseUrl:		"https://api.opsgenie.com",
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

var mockConfFileContent = []byte(`{
	"apiKey": "ApiKey",
	"baseUrl": "https://api.opsgenie.com",
    "actionMappings": {
        "Create": {
            "filePath": "/path/to/runbook.bin",
            "source": "local",
            "environmentVariables": [
                "e1=v1", "e2=v2"
            ]
        },
        "Close": {
            "source": "github",
            "repoOwner": "testAccount",
            "repoName": "testRepo",
            "repoFilePath": "marid/testConfig.json",
            "repoToken": "testToken",
            "environmentVariables": [
                "e1=v1", "e2=v2"
            ]
        }
    }
}`)

const testLocalConfFilePath = "/path/to/test/conf/file.json"

func mockReadConfigurationFromGitHub(owner, repo, filepath, token string) (*Configuration, error) {
	readConfigurationFromGitHubCalled = true

	if len(owner) <= 0 {
		return nil, errors.New("Owner was empty.")
	}

	if len(repo) <= 0 {
		return nil, errors.New("Repo was empty.")
	}

	if len(filepath) <= 0 {
		return nil, errors.New("Filepath was empty.")
	}

	if len(token) <= 0 {
		return nil, errors.New("Token was empty.")
	}

	return mockConf, nil
}

func mockReadConfigurationFromLocalWithDefaultPath(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true
	homePath, err := getHomePath()

	if err != nil {
		return nil, err
	}

	if confPath != homePath +defaultConfPath {
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

	if confPath != homePath +defaultConfPath {
		return nil, errors.New("confPath was not as the same as the default path.")
	}

	testConfMapWithoutActionMappings := &Configuration{}

	return testConfMapWithoutActionMappings, nil
}

func mockReadConfigurationFromLocalWithCustomPath(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true

	if confPath != testLocalConfFilePath {
		return nil, errors.New("confPath was not as the same as the testLocalConfFilePath.")
	}

	return mockConf, nil
}

func TestReadConfFileFromGitHub(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "github")
	os.Setenv("MARIDCONFGITHUBOWNER", "metehanozturk")
	os.Setenv("MARIDCONFGITHUBREPO", "test-repo")
	os.Setenv("MARIDCONFGITHUBFILEPATH", "marid/testConf.json")
	os.Setenv("MARIDCONFGITHUBTOKEN", "token")

	oldReadFromGitHubFunction := readConfigurationFromGitHubFunc
	defer func() { readConfigurationFromGitHubFunc = oldReadFromGitHubFunction }()
	readConfigurationFromGitHubFunc = mockReadConfigurationFromGitHub

	configuration, err := ReadConfFile()

	assert.Nil(t, err)

	assert.True(t, readConfigurationFromGitHubCalled,
		"ReadConfFile did not call the method readConfigurationFromGitHub.")
	readConfigurationFromGitHubCalled = false
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")

	assert.Equal(t, mockConf, configuration,
		"Global configuration was not equal to given configuration.")

}

func TestReadConfFileFromLocalWithDefaultPath(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")

	oldReadFromLocalFunction := readConfigurationFromLocalFunc
	defer func() { readConfigurationFromLocalFunc = oldReadFromLocalFunction }()
	readConfigurationFromLocalFunc = mockReadConfigurationFromLocalWithDefaultPath

	configuration, err := ReadConfFile()

	assert.Nil(t, err)

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitHubCalled,
		"ReadConfFile should not call the method readConfigurationFromGitHub.")
	assert.Equal(t, mockConf, configuration,
		"Global configuration was not equal to given configuration.")
}

func TestReturnErrorIfActionMappingsNotFoundInTheConfFile(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")

	oldReadFromLocalFunction := readConfigurationFromLocalFunc
	defer func() { readConfigurationFromLocalFunc = oldReadFromLocalFunction }()
	readConfigurationFromLocalFunc = mockReadConfigurationFromLocalWithDefaultPathWithoutActionMappings
	_, err := ReadConfFile()

	assert.Error(t, err, "Error should be thrown because action mappings do not exist in the configuration.")
	assert.Equal(t, "Action mappings configuration is not found in the configuration file.", err.Error(),
		"Error message was not equal to expected.")

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitHubCalled,
		"ReadConfFile should not call the method readConfigurationFromGitHub.")
}

func TestReadConfFileFromLocalWithCustomPath(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "local")
	os.Setenv("MARIDCONFLOCALFILEPATH", testLocalConfFilePath)

	oldReadFromLocalFunction := readConfigurationFromLocalFunc
	defer func() { readConfigurationFromLocalFunc = oldReadFromLocalFunction}()
	readConfigurationFromLocalFunc = mockReadConfigurationFromLocalWithCustomPath
	configuration, err := ReadConfFile()

	assert.Nil(t, err)

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitHubCalled,
		"ReadConfFile should not call the method readConfigurationFromGitHub.")
	assert.Equal(t, configuration, mockConf,
		"Global configuration was not equal to given configuration.")
}

func TestReadConfFileWithUnknownSource(t *testing.T) {
	os.Setenv("MARIDCONFSOURCE", "dummy")

	_, err := ReadConfFile()
	assert.Error(t, err, "Error should be thrown.")

	if err.Error() != "Unknown configuration source [dummy]." {
		t.Error("Error message was wrong.")
	}

	assert.False(t, readConfigurationFromGitHubCalled,
		"ReadConfFile should not call the method readConfigurationFromGitHub.")
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")
}
