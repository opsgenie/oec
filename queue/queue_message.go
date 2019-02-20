package queue

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/ois/conf"
	"github.com/opsgenie/ois/git"
	"github.com/opsgenie/ois/runbook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"time"
)

type QueueMessage interface {
	Message() *sqs.Message
	Process() (*runbook.ActionResultPayload, error)
}

type OISQueueMessage struct {
	message        *sqs.Message
	actionMappings *conf.ActionMappings
	repositories   *git.Repositories
}

func NewOISMessage(message *sqs.Message, actionMappings *conf.ActionMappings, repositories *git.Repositories) QueueMessage {

	return &OISQueueMessage{
		message:        message,
		actionMappings: actionMappings,
		repositories:   repositories,
	}
}

func (qm *OISQueueMessage) Message() *sqs.Message {
	return qm.message
}

func (qm *OISQueueMessage) Process() (*runbook.ActionResultPayload, error) {
	queuePayload := QueuePayload{}
	err := json.Unmarshal([]byte(*qm.message.Body), &queuePayload)
	if err != nil {
		return nil, err
	}

	action := queuePayload.Action
	if action == "" {
		return nil, errors.New("SQS message does not contain action property.")
	}

	mappedAction, ok := (*qm.actionMappings)[conf.ActionName(action)]
	if !ok {
		return nil, errors.Errorf("There is no mapped action found for action[%s].", action)
	}

	exePath, err := getExePath(&mappedAction, qm.repositories)
	if err != nil {
		return nil, err
	}

	result := &runbook.ActionResultPayload{
		AlertId: queuePayload.Alert.AlertId,
		Action:  queuePayload.Action,
	}

	start := time.Now()
	_, errorOutput, err := runbook.ExecuteFunc(exePath, []string{*qm.message.Body}, mappedAction.EnvironmentVariables)
	if err != nil {
		result.FailureMessage = fmt.Sprintf("Err: %s, Stderr: %s", err.Error(), errorOutput)
		logrus.Debugf("Action[%s] execution of message[%s] failed: %s Stderr: %s", action, *qm.message.MessageId, err, errorOutput)
	} else {
		result.IsSuccessful = true
		took := time.Now().Sub(start)
		logrus.Debugf("Action[%s] execution of message[%s] has been completed and it took %f seconds.", action, *qm.message.MessageId, took.Seconds())
	}

	return result, nil
}

func getExePath(mappedAction *conf.MappedAction, repositories *git.Repositories) (string, error) {
	source := mappedAction.SourceType
	exePath := mappedAction.Filepath

	switch source {
	case conf.LocalSourceType:
		return exePath, nil

	case conf.GitSourceType:
		if repositories == nil {
			return "", errors.New("Repositories should be provided.")
		}

		url := mappedAction.GitOptions.Url

		repository, err := repositories.Get(url)
		if err != nil {
			return "", err
		}

		repository.RLock()
		defer repository.RUnlock()

		exePath = filepath.Join(repository.Path, exePath)

		return exePath, nil

	default:
		return "", errors.Errorf("Unknown runbook source[%s].", source)
	}
}
