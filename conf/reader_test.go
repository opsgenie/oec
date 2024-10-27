package conf

import (
	"os"
	"testing"

	"github.com/opsgenie/oec/git"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var readFileFromGitCalled = false
var readFileFromLocalCalled = false

var mockConf = &Configuration{
	ApiKey:  "ApiKey",
	BaseUrl: "https://api.opsgenie.com",
	ActionSpecifications: ActionSpecifications{
		ActionMappings: mockActionMappings,
	},
}

var mockActionMappings = ActionMappings{
	"Create": MappedAction{
		Type:       "custom",
		SourceType: "local",
		Filepath:   "/path/to/action.bin",
		Env:        []string{"e1=v1", "e2=v2"},
	},
	"Close": MappedAction{
		Type:       "custom",
		SourceType: "git",
		GitOptions: git.Options{
			Url:                "testUrl",
			PrivateKeyFilepath: "testKeyPath",
			Branch:             "main",
		},
		Env:      []string{"e1=v1", "e2=v2"},
		Filepath: "/path/to/action.bin",
	},
	"WithHttpAction": MappedAction{
		Type:       "http",
		SourceType: "local",
		Filepath:   "/path/to/http-executor",
		Flags: Flags{
			"url":     "https://opsgenie.com",
			"headers": "{\"Authentication\":\"Basic JNjDkNsKaMs\"}",
			"params":  "{\"Key1\":\"Value1\"}",
			"method":  "PUT",
		},
	},
}

func expectedConf() *Configuration {
	expectedConf := *mockConf
	expectedConf.LogLevel = "info"
	expectedConf.ActionMappings = copyActionMappings(mockConf.ActionMappings)
	addHomeDirPrefixToActionMappings(expectedConf.ActionMappings)
	expectedConf.GlobalArgs = append([]string{
		"-apiKey", expectedConf.ApiKey,
		"-opsgenieUrl", expectedConf.BaseUrl,
		"-logLevel", "INFO"},
		expectedConf.GlobalArgs...,
	)

	if expectedConf.LogrusLevel == 0 {
		expectedConf.LogrusLevel = logrus.InfoLevel
	}
	return &expectedConf
}

var mockJsonFileContent = []byte(`{
	"apiKey": "ApiKey",
	"baseUrl": "https://api.opsgenie.com",
    "actionMappings": {
        "Create": {
            "filepath": "/path/to/action.bin",
            "sourceType": "local",
            "env": [
                "e1=v1", "e2=v2"
            ]
        },
        "Close": {
            "sourceType": "git",
            "gitOptions" : {
                "url" : "testUrl",
                "privateKeyFilepath" : "testKeyPath",
                "branch" : "main"
            },
            "env": [
                "e1=v1", "e2=v2"
            ],
			"filepath": "/path/to/action.bin"
        },
		"WithHttpAction" : {
			"type" : "http",
            "filepath": "/path/to/http-executor",
			"url": "https://opsgenie.com",
			"headers": {
				"Authentication": "Basic JNjDkNsKaMs"
			},
			"params": {
				"Key1" : "Value1"
			},
			"method" : "PUT",
            "sourceType": "local"
        }
    }
}`)

var mockYamlFileContent = []byte(`
---
apiKey: ApiKey
baseUrl: https://api.opsgenie.com
actionMappings:
  Create:
    filepath: "/path/to/action.bin"
    sourceType: local
    env:
    - e1=v1
    - e2=v2
  Close:
    sourceType: git
    gitOptions:
      url: testUrl
      privateKeyFilepath: testKeyPath
      branch: main
    env:
    - e1=v1
    - e2=v2
    filepath: "/path/to/action.bin"
  WithHttpAction:
    type: "http"
    filepath: "/path/to/http-executor"
    url: "https://opsgenie.com"
    headers:
      Authentication: "Basic JNjDkNsKaMs"
    params:
      Key1: Value1
    method: PUT
    sourceType: local
`)

const testLocalConfFilePath = "/path/to/test/conf/file.json"

func mockReadFileFromGit(owner, repo, filepath, token, branch string) (*Configuration, error) {
	readFileFromGitCalled = true

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

	if len(branch) <= 0 {
		return nil, errors.New("Branch was empty.")
	}

	conf := *mockConf
	conf.ActionMappings = copyActionMappings(mockActionMappings)

	return &conf, nil
}

func mockReadFileFromLocalDefaultPath(confPath string) (*Configuration, error) {
	readFileFromLocalCalled = true

	if confPath != addHomeDirPrefix(defaultConfFilepath) {
		return nil, errors.Errorf("confPath was not as the same as the default path, confPath[%s] != default[%s]", confPath, addHomeDirPrefix(confPath))
	}

	conf := *mockConf
	conf.ActionMappings = copyActionMappings(mockActionMappings)

	return &conf, nil
}

func mockReadFileFromLocalWithoutActionMappings(confPath string) (*Configuration, error) {
	readFileFromLocalCalled = true

	if confPath != addHomeDirPrefix(defaultConfFilepath) {
		return nil, errors.Errorf("confPath was not as the same as the default path, confPath[%s] != default[%s]", confPath, addHomeDirPrefix(confPath))
	}

	testConfMapWithoutActionMappings := &Configuration{
		ApiKey:  "ApiKey",
		BaseUrl: "BaseUrl",
	}

	return testConfMapWithoutActionMappings, nil
}

func mockReadFileFromLocalCustomPath(confPath string) (*Configuration, error) {
	readFileFromLocalCalled = true

	if confPath != addHomeDirPrefix(testLocalConfFilePath) {
		return nil, errors.Errorf("confPath was not as the same as the testLocalConfFilePath, confPath[%s] != testPath[%s]", confPath, addHomeDirPrefix(confPath))
	}

	conf := *mockConf
	conf.ActionMappings = copyActionMappings(mockActionMappings)

	return &conf, nil
}

func TestRead(t *testing.T) {

	t.Run("TestReadFileFromGit", testReadFileFromGit)
	t.Run("TestReadFileFromLocalDefaultPath", testReadFileFromLocalDefaultPath)
	t.Run("TestReadFileWithoutActionMappings", testReadFileWithoutActionMappings)
	t.Run("TestReadFileFromLocalCustomPath", testReadFileFromLocalCustomPath)

	readFileFromGitFunc = readFileFromGit
}

func testReadFileFromGit(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE_TYPE", "git")
	os.Setenv("OEC_CONF_GIT_URL", "utl")
	os.Setenv("OEC_CONF_GIT_PRIVATE_KEY_FILEPATH", "/test_id_rsa")
	os.Setenv("OEC_CONF_GIT_FILEPATH", "oec/testConf.json")
	os.Setenv("OEC_CONF_GIT_PASSPHRASE", "pass")
	os.Setenv("OEC_CONF_GIT_BRANCH", "main")

	readFileFromGitFunc = mockReadFileFromGit
	configuration, err := Read()

	assert.Nil(t, err)

	assert.True(t, readFileFromGitCalled,
		"Read method did not call the method readFileFromGit.")
	readFileFromGitCalled = false
	assert.False(t, readFileFromLocalCalled,
		"Read method should not call the method readFileFromLocal.")

	assert.Equal(t, expectedConf(), configuration,
		"Global configuration was not equal to given configuration.")

}

func testReadFileFromLocalDefaultPath(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE_TYPE", "local")

	readFileFromLocalFunc = mockReadFileFromLocalDefaultPath
	configuration, err := Read()

	assert.Nil(t, err)

	assert.True(t, readFileFromLocalCalled,
		"Read method did not call the method readFileFromLocal.")
	readFileFromLocalCalled = false
	assert.False(t, readFileFromGitCalled,
		"Read method should not call the method readFileFromGit.")
	assert.Equal(t, expectedConf(), configuration,
		"Global configuration was not equal to given configuration.")
}

func testReadFileWithoutActionMappings(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE_TYPE", "local")

	readFileFromLocalFunc = mockReadFileFromLocalWithoutActionMappings
	_, err := Read()

	assert.Error(t, err, "Error should be thrown because action mappings do not exist in the configuration.")
	assert.Equal(t, "Action mappings configuration is not found in the configuration file.", err.Error(),
		"Error message was not equal to expected.")

	assert.True(t, readFileFromLocalCalled,
		"Read method did not call the method readFileFromLocal.")
	readFileFromLocalCalled = false
	assert.False(t, readFileFromGitCalled,
		"Read method should not call the method readFileFromGit.")
}

func testReadFileFromLocalCustomPath(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE_TYPE", "local")
	os.Setenv("OEC_CONF_LOCAL_FILEPATH", testLocalConfFilePath)

	readFileFromLocalFunc = mockReadFileFromLocalCustomPath
	configuration, err := Read()

	assert.Nil(t, err)

	assert.True(t, readFileFromLocalCalled,
		"Read method did not call the method readFileFromLocal.")
	readFileFromLocalCalled = false
	assert.False(t, readFileFromGitCalled,
		"Read method should not call the method readFileFromGit.")
	assert.Equal(t, expectedConf(), configuration,
		"Global configuration was not equal to given configuration.")
}

func TestReadFileFromUnknownSource(t *testing.T) {
	os.Setenv("OEC_CONF_SOURCE_TYPE", "dummy")

	_, err := Read()
	assert.Error(t, err, "Error should be thrown.")

	assert.Equal(t, "Unknown configuration source type[dummy], valid types are \"local\" and \"git\".", err.Error())

	assert.False(t, readFileFromGitCalled,
		"Read method should not call the method readFileFromGit.")
	assert.False(t, readFileFromLocalCalled,
		"Read method should not call the method readFileFromLocal.")
}
