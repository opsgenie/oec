package conf

import (
	"github.com/opsgenie/oec/git"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var readConfigurationFromGitCalled = false
var readConfigurationFromLocalCalled = false

var mockConf = &Configuration{
	ApiKey:  "ApiKey",
	BaseUrl: "https://api.opsgenie.com",
	ActionSpecifications: ActionSpecifications{
		ActionMappings: mockActionMappings,
	},
}

var mockActionMappings = ActionMappings{
	"Create": MappedAction{
		SourceType: "local",
		Filepath:   "/path/to/runbook.bin",
		Env:        []string{"e1=v1", "e2=v2"},
	},
	"Close": MappedAction{
		SourceType: "git",
		GitOptions: git.GitOptions{
			Url:                "testUrl",
			PrivateKeyFilepath: "testKeyPath",
		},
		Env:      []string{"e1=v1", "e2=v2"},
		Filepath: "/path/to/runbook.bin",
	},
}

func expectedConf() *Configuration {
	expectedConf := *mockConf
	expectedConf.ActionMappings = copyActionMappings(mockConf.ActionMappings)
	addHomeDirPrefixToActionMappings(expectedConf.ActionMappings)
	expectedConf.GlobalArgs = append([]string{"-apiKey", expectedConf.ApiKey, "-opsgenieUrl", expectedConf.BaseUrl}, expectedConf.GlobalArgs...)

	if expectedConf.LogrusLevel == 0 {
		expectedConf.LogrusLevel = logrus.InfoLevel
	}
	return &expectedConf
}

var mockJsonConfFileContent = []byte(`{
	"apiKey": "ApiKey",
	"baseUrl": "https://api.opsgenie.com",
    "actionMappings": {
        "Create": {
            "filepath": "/path/to/runbook.bin",
            "sourceType": "local",
            "env": [
                "e1=v1", "e2=v2"
            ]
        },
        "Close": {
            "sourceType": "git",
            "gitOptions" : {
                "url" : "testUrl",
                "privateKeyFilepath" : "testKeyPath"
            },
            "env": [
                "e1=v1", "e2=v2"
            ],
			"filepath": "/path/to/runbook.bin"
        }
    }
}`)

var mockYamlConfFileContent = []byte(`
---
apiKey: ApiKey
baseUrl: https://api.opsgenie.com
actionMappings:
  Create:
    filepath: "/path/to/runbook.bin"
    sourceType: local
    env:
    - e1=v1
    - e2=v2
  Close:
    sourceType: git
    gitOptions:
      url: testUrl
      privateKeyFilepath: testKeyPath
    env:
    - e1=v1
    - e2=v2
    filepath: "/path/to/runbook.bin"
`)

const testLocalConfFilePath = "/path/to/test/conf/file.json"

func mockReadConfigurationFromGit(owner, repo, filepath, token string) (*Configuration, error) {
	readConfigurationFromGitCalled = true

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

	conf := *mockConf
	conf.ActionMappings = copyActionMappings(mockActionMappings)

	return &conf, nil
}

func mockReadConfigurationFromLocalWithDefaultPath(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true

	if confPath != addHomeDirPrefix(defaultConfFilepath) {
		return nil, errors.Errorf("confPath was not as the same as the default path, confPath[%s] != default[%s]", confPath, addHomeDirPrefix(confPath))
	}

	conf := *mockConf
	conf.ActionMappings = copyActionMappings(mockActionMappings)

	return &conf, nil
}

func mockReadConfigurationFromLocalWithDefaultPathWithoutActionMappings(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true

	if confPath != addHomeDirPrefix(defaultConfFilepath) {
		return nil, errors.Errorf("confPath was not as the same as the default path, confPath[%s] != default[%s]", confPath, addHomeDirPrefix(confPath))
	}

	testConfMapWithoutActionMappings := &Configuration{
		ApiKey:  "ApiKey",
		BaseUrl: "BaseUrl",
	}

	return testConfMapWithoutActionMappings, nil
}

func mockReadConfigurationFromLocalWithCustomPath(confPath string) (*Configuration, error) {
	readConfigurationFromLocalCalled = true

	if confPath != addHomeDirPrefix(testLocalConfFilePath) {
		return nil, errors.Errorf("confPath was not as the same as the testLocalConfFilePath, confPath[%s] != testPath[%s]", confPath, addHomeDirPrefix(confPath))
	}

	conf := *mockConf
	conf.ActionMappings = copyActionMappings(mockActionMappings)

	return &conf, nil
}

func TestReadConfFile(t *testing.T) {

	t.Run("TestReadConfFileFromGit", testReadConfFileFromGit)
	t.Run("TestReadConfFileFromLocalWithDefaultPath", testReadConfFileFromLocalWithDefaultPath)
	t.Run("TestReadConfFileWithoutActionMappings", testReadConfFileWithoutActionMappings)
	t.Run("TestReadConfFileFromLocalWithCustomPath", testReadConfFileFromLocalWithCustomPath)

	readConfigurationFromGitFunc = readConfigurationFromGit
}

func testReadConfFileFromGit(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE", "git")
	os.Setenv("OEC_CONF_GIT_URL", "utl")
	os.Setenv("OEC_CONF_GIT_PRIVATE_KEY_FILEPATH", "/test_id_rsa")
	os.Setenv("OEC_CONF_GIT_FILEPATH", "oec/testConf.json")
	os.Setenv("OEC_CONF_GIT_PASSPHRASE", "pass")

	readConfigurationFromGitFunc = mockReadConfigurationFromGit
	configuration, err := ReadConfFile()

	assert.Nil(t, err)

	assert.True(t, readConfigurationFromGitCalled,
		"ReadConfFile did not call the method readConfigurationFromGit.")
	readConfigurationFromGitCalled = false
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")

	assert.Equal(t, expectedConf(), configuration,
		"Global configuration was not equal to given configuration.")

}

func testReadConfFileFromLocalWithDefaultPath(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE", "local")

	readConfigurationFromLocalFunc = mockReadConfigurationFromLocalWithDefaultPath
	configuration, err := ReadConfFile()

	assert.Nil(t, err)

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.Equal(t, expectedConf(), configuration,
		"Global configuration was not equal to given configuration.")
}

func testReadConfFileWithoutActionMappings(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE", "local")

	readConfigurationFromLocalFunc = mockReadConfigurationFromLocalWithDefaultPathWithoutActionMappings
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

func testReadConfFileFromLocalWithCustomPath(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE", "local")
	os.Setenv("OEC_CONF_LOCAL_FILEPATH", testLocalConfFilePath)

	readConfigurationFromLocalFunc = mockReadConfigurationFromLocalWithCustomPath
	configuration, err := ReadConfFile()

	assert.Nil(t, err)

	assert.True(t, readConfigurationFromLocalCalled,
		"ReadConfFile did not call the method readConfigurationFromLocal.")
	readConfigurationFromLocalCalled = false
	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.Equal(t, expectedConf(), configuration,
		"Global configuration was not equal to given configuration.")
}

func TestReadConfFileWithUnknownSource(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE", "dummy")

	_, err := ReadConfFile()
	assert.Error(t, err, "Error should be thrown.")

	if err.Error() != "Unknown configuration source[dummy]." {
		t.Error("Error message was wrong.")
	}

	assert.False(t, readConfigurationFromGitCalled,
		"ReadConfFile should not call the method readConfigurationFromGit.")
	assert.False(t, readConfigurationFromLocalCalled,
		"ReadConfFile should not call the method readConfigurationFromLocal.")
}
