package queue

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/git"
	"github.com/opsgenie/marid2/runbook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
		return nil, errors.New("SQS message does not contain action property")
	}

	mappedAction, ok := (*qm.actionMappings)[conf.ActionName(action)]
	if !ok {
		return nil, errors.Errorf("There is no mapped action found for [%s]", action)
	}

	_, errorOutput, err := runbook.ExecuteRunbookFunc(&mappedAction, qm.repositories, []string{*qm.message.Body})
	if err != nil {
		return nil, errors.Errorf("Action[%s] execution of message[%s] failed: %s", action, *qm.message.MessageId, err)
	}

	var success bool
	if errorOutput != "" {
		logrus.Debugf("Action[%s] execution of message[%s] produced error output: %s", action, *qm.message.MessageId, errorOutput)
	} else {
		success = true
		logrus.Debugf("Action[%s] execution of message[%s] has been completed.", action, *qm.message.MessageId)
	}

	result := &runbook.ActionResultPayload{
		IsSuccessful:   success,
		AlertId:        queuePayload.Alert.AlertId,
		Action:         queuePayload.Action,
		FailureMessage: errorOutput,
	}

	return result, nil
}

func NewOISMessage(message *sqs.Message, actionMappings *conf.ActionMappings, repositories *git.Repositories) QueueMessage {

	return &OISQueueMessage{
		message:        message,
		actionMappings: actionMappings,
		repositories:   repositories,
	}
}
