package queue

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/runbook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type QueueMessage interface {
	Message() *sqs.Message
	Process() (*runbook.ActionResultPayload ,error)
}

type MaridQueueMessage struct {
	message 		*sqs.Message
	actionMappings 	*conf.ActionMappings
}

func (mqm *MaridQueueMessage) Message() *sqs.Message {
	return mqm.message
}

func (mqm *MaridQueueMessage) Process() (*runbook.ActionResultPayload ,error) {
	queuePayload := QueuePayload{}
	err := json.Unmarshal([]byte(*mqm.message.Body), &queuePayload)
	if err != nil {
		return nil, err
	}

	action := queuePayload.Action
	if action == "" {
		return nil, errors.New("SQS message does not contain action property")
	}

	mappedAction, ok := (map[conf.ActionName]conf.MappedAction)(*mqm.actionMappings)[conf.ActionName(action)]
	if !ok {
		return nil, errors.Errorf("There is no mapped action found for [%s]", action)
	}

	_, errorOutput, err := runbook.ExecuteRunbookFunc(&mappedAction, *mqm.message.Body)
	if err != nil {
		return nil, errors.Errorf("Action[%s] execution of message[%s] failed: %s", action, *mqm.message.MessageId, err)
	}

	var success bool
	if errorOutput != "" {
		logrus.Debugf("Action[%s] execution of message[%s] produced error output: %s", action, *mqm.message.MessageId, errorOutput)
	} else {
		success = true
		logrus.Debugf("Action[%s] execution of message[%s] has been completed.", action, *mqm.message.MessageId)
	}

	result := &runbook.ActionResultPayload {
		IsSuccessful:   success,
		AlertId:        queuePayload.Alert.AlertId,
		Action:         queuePayload.Action,
		FailureMessage: errorOutput,
	}

	return result, nil
}

func NewMaridMessage(message *sqs.Message, actionMappings *conf.ActionMappings) QueueMessage {

	return &MaridQueueMessage{
		message: 		message,
		actionMappings:	actionMappings,
	}
}
