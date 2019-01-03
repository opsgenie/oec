package conf

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

type Configuration struct {
	ApiKey 			string 			`json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	BaseUrl 		string			`json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	ActionMappings 	ActionMappings 	`json:"actionMappings,omitempty"`
	PollerConf 		PollerConf 		`json:"pollerConf,omitempty"`
	PoolConf 		PoolConf 		`json:"poolConf,omitempty"`
	LogLevel		logrus.Level	`json:"logLevel,omitempty"`
}

type ActionName string

type ActionMappings map[ActionName]MappedAction

type MappedAction struct {
	Source               string   `json:"source,omitempty"`
	RepoOwner            string   `json:"repoOwner,omitempty"`
	RepoName             string   `json:"repoName,omitempty"`
	RepoFilePath         string   `json:"repoFilePath,omitempty"`
	RepoToken            string   `json:"repoToken,omitempty"`
	FilePath             string   `json:"filePath,omitempty"`
	EnvironmentVariables []string `json:"environmentVariables,omitempty"`
}

type PollerConf struct {
	PollingWaitIntervalInMillis time.Duration `json:"pollingWaitIntervalInMillis,omitempty"`
	VisibilityTimeoutInSeconds  int64         `json:"visibilityTimeoutInSeconds,omitempty"`
	MaxNumberOfMessages         int64         `json:"maxNumberOfMessages,omitempty"`
}

type PoolConf struct {
	MaxNumberOfWorker        int32			`json:"maxNumberOfWorker,omitempty"`
	MinNumberOfWorker        int32			`json:"minNumberOfWorker,omitempty"`
	QueueSize                int32			`json:"queueSize,omitempty"`
	KeepAliveTimeInMillis    time.Duration	`json:"keepAliveTimeInMillis,omitempty"`
	MonitoringPeriodInMillis time.Duration	`json:"monitoringPeriodInMillis,omitempty"`
}

var readConfigurationFromGitHubFunc = readConfigurationFromGitHub
var readConfigurationFromLocalFunc = readConfigurationFromLocal

var defaultConfPath = strings.Join([]string{"opsgenie", "maridConfig.json"}, string(os.PathSeparator))

func ReadConfFile() (*Configuration, error) {

	confSource := os.Getenv("MARID_CONF_SOURCE")
	conf, err := readConfFileFromSource(strings.ToLower(confSource))

	if err != nil {
		return nil, err
	}
	if len(conf.ActionMappings) == 0 {
		return nil, errors.New("Action mappings configuration is not found in the configuration file.")
	}
	if conf.ApiKey == "" {
		return nil, errors.New("ApiKey is not found in the configuration file.")
	}
	if conf.BaseUrl == "" {
		return nil, errors.New("BaseUrl is not found in the configuration file.")
	}

	return conf, nil
}

func readConfFileFromSource(confSource string) (*Configuration, error) {

	switch confSource {
	case "github":
		owner := os.Getenv("MARID_CONF_GITHUB_OWNER")
		repo := os.Getenv("MARID_CONF_GITHUB_REPO")
		filepath := os.Getenv("MARID_CONF_GITHUB_FILEPATH")
		token := os.Getenv("MARID_CONF_GITHUB_TOKEN")

		return readConfigurationFromGitHubFunc(owner, repo, filepath, token)
	case "local":
		maridConfPath := os.Getenv("MARID_CONF_LOCAL_FILEPATH")

		if len(maridConfPath) <= 0 {
			homePath, err := getHomePath()
			if err != nil {
				return nil, err
			}

			maridConfPath = strings.Join([]string{homePath, defaultConfPath}, string(os.PathSeparator))
		}

		return readConfigurationFromLocalFunc(maridConfPath)
	default:
		return nil, errors.Errorf("Unknown configuration source [%s].", confSource)
	}
}
