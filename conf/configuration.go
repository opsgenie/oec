package conf

import (
	"github.com/opsgenie/oec/git"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	LocalSourceType = "local"
	GitSourceType   = "git"

	DefaultBaseUrl = "https://api.opsgenie.com"
)

var readConfigurationFromGitFunc = readConfigurationFromGit
var readConfigurationFromLocalFunc = readConfigurationFromLocal

var defaultConfFilepath = filepath.Join("~", "oec", "config.json")

type Configuration struct {
	ActionSpecifications `yaml:",inline"`
	AppName              string     `json:"appName" yaml:"appName"`
	ApiKey               string     `json:"apiKey" yaml:"apiKey"`
	BaseUrl              string     `json:"baseUrl" yaml:"baseUrl"`
	PollerConf           PollerConf `json:"pollerConf" yaml:"pollerConf"`
	PoolConf             PoolConf   `json:"poolConf" yaml:"poolConf"`
	LogLevel             string     `json:"logLevel" yaml:"logLevel"`
	LogrusLevel          logrus.Level
}

type ActionSpecifications struct {
	ActionMappings ActionMappings `json:"actionMappings" yaml:"actionMappings"`
	GlobalFlags    Flags          `json:"globalFlags" yaml:"globalFlags"`
	GlobalArgs     []string       `json:"globalArgs" yaml:"globalArgs"`
	GlobalEnv      []string       `json:"globalEnv" yaml:"globalEnv"`
}

type ActionName string

type ActionMappings map[ActionName]MappedAction

func (m ActionMappings) GitActions() []git.GitOptions {

	opts := make([]git.GitOptions, 0)
	for _, action := range m {
		if (action.GitOptions != git.GitOptions{}) {
			opts = append(opts, action.GitOptions)
		}
	}

	return opts
}

type MappedAction struct {
	SourceType string         `json:"sourceType" yaml:"sourceType"`
	GitOptions git.GitOptions `json:"gitOptions" yaml:"gitOptions"`
	Filepath   string         `json:"filepath" yaml:"filepath"`
	Flags      Flags          `json:"flags" yaml:"flags"`
	Args       []string       `json:"args" yaml:"args"`
	Env        []string       `json:"env" yaml:"env"`
	Stdout     string         `json:"stdout" yaml:"stdout"`
	Stderr     string         `json:"stderr" yaml:"stderr"`
}

type Flags map[string]string

func (f Flags) Args() []string {

	args := make([]string, 0)
	for flagName, flagValue := range f {
		args = append(args, "-"+flagName)
		args = append(args, flagValue)
	}

	return args
}

type PollerConf struct {
	PollingWaitIntervalInMillis time.Duration `json:"pollingWaitIntervalInMillis" yaml:"pollingWaitIntervalInMillis"`
	VisibilityTimeoutInSeconds  int64         `json:"visibilityTimeoutInSeconds" yaml:"visibilityTimeoutInSeconds"`
	MaxNumberOfMessages         int64         `json:"maxNumberOfMessages" yaml:"maxNumberOfMessages"`
}

type PoolConf struct {
	MaxNumberOfWorker        int32         `json:"maxNumberOfWorker" yaml:"maxNumberOfWorker"`
	MinNumberOfWorker        int32         `json:"minNumberOfWorker" yaml:"minNumberOfWorker"`
	QueueSize                int32         `json:"queueSize" yaml:"queueSize"`
	KeepAliveTimeInMillis    time.Duration `json:"keepAliveTimeInMillis" yaml:"keepAliveTimeInMillis"`
	MonitoringPeriodInMillis time.Duration `json:"monitoringPeriodInMillis" yaml:"monitoringPeriodInMillis"`
}

func ReadConfFile() (*Configuration, error) {

	confSourceType := os.Getenv("OEC_CONF_SOURCE_TYPE")
	conf, err := readConfFileFromSource(strings.ToLower(confSourceType))
	if err != nil {
		return nil, err
	}

	err = validateConfiguration(conf)
	if err != nil {
		return nil, err
	}

	addHomeDirPrefixToActionMappings(conf.ActionMappings)
	chmodLocalActions(conf.ActionMappings, 0700)

	conf.addDefaultFlags()

	return conf, nil
}

func (c *Configuration) addDefaultFlags() {
	c.GlobalArgs = append(
		[]string{
			"-apiKey", c.ApiKey,
			"-opsgenieUrl", c.BaseUrl,
			"-logLevel", strings.ToUpper(c.LogLevel),
		},
		c.GlobalArgs...,
	)
}

func validateConfiguration(conf *Configuration) error {

	if conf == nil || conf == (&Configuration{}) {
		return errors.New("The configuration is empty.")
	}
	if conf.ApiKey == "" {
		return errors.New("ApiKey is not found in the configuration file.")
	}
	if conf.BaseUrl == "" {
		conf.BaseUrl = DefaultBaseUrl
		logrus.Infof("BaseUrl is not found in the configuration file, default url[%s] is set.", DefaultBaseUrl)
	}

	if len(conf.ActionMappings) == 0 {
		return errors.New("Action mappings configuration is not found in the configuration file.")
	} else {
		for actionName, action := range conf.ActionMappings {
			if action.SourceType != LocalSourceType &&
				action.SourceType != GitSourceType {
				return errors.Errorf("Action source type of action[%s] should be either local or git.", actionName)
			} else {
				if action.Filepath == "" {
					return errors.Errorf("Filepath of action[%s] is empty.", actionName)
				}
				if action.SourceType == GitSourceType &&
					action.GitOptions == (git.GitOptions{}) {
					return errors.Errorf("Git options of action[%s] is empty.", actionName)
				}
			}
		}
	}

	level, err := logrus.ParseLevel(conf.LogLevel)
	if err != nil {
		conf.LogrusLevel = logrus.InfoLevel
		conf.LogLevel = "info"
	} else {
		conf.LogrusLevel = level
	}

	return nil
}

func readConfFileFromSource(confSourceType string) (*Configuration, error) {

	switch confSourceType {
	case GitSourceType:
		url := os.Getenv("OEC_CONF_GIT_URL")
		privateKeyFilepath := os.Getenv("OEC_CONF_GIT_PRIVATE_KEY_FILEPATH")
		passphrase := os.Getenv("OEC_CONF_GIT_PASSPHRASE")
		confFilepath := os.Getenv("OEC_CONF_GIT_FILEPATH")

		if privateKeyFilepath != "" {
			privateKeyFilepath = addHomeDirPrefix(privateKeyFilepath)
		}

		if confFilepath == "" {
			return nil, errors.New("Git configuration filepath could not be empty.")
		}

		return readConfigurationFromGitFunc(url, privateKeyFilepath, passphrase, confFilepath)
	case LocalSourceType:
		confFilepath := os.Getenv("OEC_CONF_LOCAL_FILEPATH")

		if len(confFilepath) <= 0 {
			confFilepath = addHomeDirPrefix(defaultConfFilepath)
		} else {
			confFilepath = addHomeDirPrefix(confFilepath)
		}

		return readConfigurationFromLocalFunc(confFilepath)
	case "":
		return nil, errors.Errorf("OEC_CONF_SOURCE_TYPE should be set as \"local\" or \"git\".")
	default:
		return nil, errors.Errorf("Unknown configuration source type[%s], valid types are \"local\" and \"git\".", confSourceType)
	}
}
