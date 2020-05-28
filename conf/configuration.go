package conf

import (
	"github.com/opsgenie/oec/git"
	"github.com/sirupsen/logrus"
	"time"
)

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

func (m ActionMappings) GitActions() []git.Options {

	opts := make([]git.Options, 0)
	for _, action := range m {
		if (action.GitOptions != git.Options{}) {
			opts = append(opts, action.GitOptions)
		}
	}

	return opts
}

type MappedAction struct {
	SourceType string      `json:"sourceType" yaml:"sourceType"`
	GitOptions git.Options `json:"gitOptions" yaml:"gitOptions"`
	Filepath   string      `json:"filepath" yaml:"filepath"`
	Flags      Flags       `json:"flags" yaml:"flags"`
	Args       []string    `json:"args" yaml:"args"`
	Env        []string    `json:"env" yaml:"env"`
	Stdout     string      `json:"stdout" yaml:"stdout"`
	Stderr     string      `json:"stderr" yaml:"stderr"`
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
