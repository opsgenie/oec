package conf

import (
	"encoding/json"
	"github.com/opsgenie/oec/git"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/url"
	"strings"
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
	Type       string      `json:"type" yaml:"type"`
	SourceType string      `json:"sourceType" yaml:"sourceType"`
	GitOptions git.Options `json:"gitOptions" yaml:"gitOptions"`
	Filepath   string      `json:"filepath" yaml:"filepath"`
	Flags      Flags       `json:"flags" yaml:"flags"`
	Args       []string    `json:"args" yaml:"args"`
	Env        []string    `json:"env" yaml:"env"`
	Stdout     string      `json:"stdout" yaml:"stdout"`
	Stderr     string      `json:"stderr" yaml:"stderr"`
}
type httpFields struct {
	Url     string            `json:"url" yaml:"url"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Params  map[string]string `json:"params" yaml:"params"`
	Method  string            `json:"method" yaml:"method"`
}

func appendHttpFields(action *MappedAction, fields httpFields) error {
	action.Flags = map[string]string{}
	if fields.Url != "" {
		action.Flags["url"] = fields.Url
	}
	if fields.Method != "" {
		action.Flags["method"] = fields.Method
	}
	if len(fields.Headers) > 0 {
		headers, err := json.Marshal(fields.Headers)
		if err != nil {
			return err
		}
		action.Flags["headers"] = string(headers)
	}
	if len(fields.Params) > 0 {
		params, err := json.Marshal(fields.Params)
		if err != nil {
			return err
		}
		action.Flags["params"] = string(params)
	}
	return nil
}

func validateHttpFields(fields httpFields) error {
	methods := map[string]bool{"GET": true, "HEAD": true, "POST": true, "PUT": true, "PATCH": true,
		"DELETE": true, "CONNECT": true, "OPTIONS": true, "TRACE": true}
	if fields.Method != "" && !methods[strings.ToUpper(fields.Method)] {
		return errors.New("Http method is not valid: [" + fields.Method + "].")
	}
	if _, err := url.Parse(fields.Url); err != nil {
		return err
	}
	return nil
}

func (action *MappedAction) UnmarshalJSON(b []byte) error {
	type mappedAction MappedAction
	err := json.Unmarshal(b, (*mappedAction)(action))
	if err != nil {
		return err
	}
	if action.Type == "http" {
		fields := httpFields{}
		if err = json.Unmarshal(b, &fields); err != nil {
			return err
		}
		if err := validateHttpFields(fields); err != nil {
			return err
		}
		if err = appendHttpFields(action, fields); err != nil {
			return err
		}
	} else if action.Type == "" {
		action.Type = "custom"
	}
	return nil
}

func (action *MappedAction) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type mappedAction MappedAction
	err := unmarshal((*mappedAction)(action))
	if err != nil {
		return err
	}
	if action.Type == "http" {
		fields := httpFields{}
		if err = unmarshal(&fields); err != nil {
			return err
		}
		if err := validateHttpFields(fields); err != nil {
			return err
		}
		if err = appendHttpFields(action, fields); err != nil {
			return err
		}
	} else if action.Type == "" {
		action.Type = "custom"
	}

	return nil
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
