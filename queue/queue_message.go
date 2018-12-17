package queue

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/runbook"
	"github.com/pkg/errors"
	"log"
)

type QueueMessage interface {
	Message() *sqs.Message
	Process() error
}

type MaridQueueMessage struct {
	message 		*sqs.Message
	actionMappings 	*conf.ActionMappings
	apiKey 			*string
	baseUrl 		*string
}

func (mqm *MaridQueueMessage) Message() *sqs.Message {
	return mqm.message
}

func (mqm *MaridQueueMessage) Process() error {
	queuePayload := QueuePayload{}
	err := json.Unmarshal([]byte(*mqm.message.Body), &queuePayload)
	if err != nil {
		return err
	}

	action := queuePayload.Action
	if action == "" {
		return errors.New("SQS message does not contain action property")
	}

	mappedAction, ok := (map[conf.ActionName]conf.MappedAction)(*mqm.actionMappings)[conf.ActionName(action)]
	if !ok {
		return errors.Errorf("There is no mapped action found for [%s]", action)
	}

	commandOutput, errorOutput, err := runbook.ExecuteRunbookFunc(&mappedAction, *mqm.message.Body)
	log.Println(commandOutput, errorOutput, err)

	var success bool
	if errorOutput == "" {
		success = true
	}

	result := &runbook.ActionResultPayload{
		IsSuccessful:   success,
		AlertId:        queuePayload.Alert.AlertId,
		Action:         queuePayload.Action,
		FailureMessage: errorOutput,

	}
	runbook.SendResultToOpsGenie(result, mqm.apiKey, mqm.baseUrl)
	return nil
}

func NewMaridMessage(message *sqs.Message, actionMappings *conf.ActionMappings, apiKey *string, baseUrl *string) QueueMessage {

	if message == nil || actionMappings == nil || apiKey == nil || baseUrl == nil {
		return nil
	}

	return &MaridQueueMessage{
		message: 		message,
		actionMappings:	actionMappings,
		apiKey:			apiKey,
		baseUrl:		baseUrl,
	}
}
