package conf

import (
	"github.com/pkg/errors"
	"os"
	"time"
	"github.com/sirupsen/logrus"
)

type Configuration struct {
	ApiKey 			string 			`json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	ActionMappings 	ActionMappings 	`json:"actionMappings,omitempty"`
	PollerConf 		PollerConf 		`json:"pollerConfcal"`
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

var readConfigurationFromGitFunction = readConfigurationFromGit
var readConfigurationFromLocalFunction = readConfigurationFromLocal

func ReadConfFile() (*Configuration, error) {

	confSource := os.Getenv("MARIDCONFSOURCE")
	conf, err := readConfFileFromSource(confSource)

	if err != nil {
		return nil, err
	}
	if len(conf.ActionMappings) == 0 {
		return nil, errors.New("Action mappings configuration is not found in the configuration file.")
	}
	if conf.ApiKey == "" {
		return nil, errors.New("ApiKey is not found in the configuration file.")
	}

	return conf, nil
}

func readConfFileFromSource(confSource string) (*Configuration, error) {
	if confSource == "git" {
		privateKeyFilePath := os.Getenv("MARIDCONFREPOPRIVATEKEYPATH")
		passPhrase := os.Getenv("MARIDCONFGITPASSPHRASE")
		gitUrl := os.Getenv("MARIDCONFREPOGITURL")
		maridConfPath := os.Getenv("MARIDCONFGITFILEPATH")

		return readConfigurationFromGitFunction(gitUrl, maridConfPath, privateKeyFilePath, passPhrase)

	} else if confSource == "local" {
		maridConfPath := os.Getenv("MARIDCONFLOCALFILEPATH")

		if len(maridConfPath) <= 0 {
			homePath, err := getHomePath()

			if err != nil {
				return nil, err
			}

			maridConfPath = homePath + string(os.PathSeparator) + ".opsgenie" + string(os.PathSeparator) + "maridConfig.json"
		}

		return readConfigurationFromLocalFunction(maridConfPath)
	} else {
		return nil, errors.New("Unknown configuration source [" + confSource + "].")
	}
}
