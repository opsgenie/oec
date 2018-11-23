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
	GetMessage() *sqs.Message
	Process() error
}

type MaridQueueMessage struct {
	*sqs.Message

	GetMessageMethod func(mqm *MaridQueueMessage) *sqs.Message
	ProcessMethod    func(mqm *MaridQueueMessage) error
}

func GetMessage(mqm *MaridQueueMessage) *sqs.Message {
	return mqm.Message
}

func (mqm *MaridQueueMessage) GetMessage() *sqs.Message {
	return mqm.GetMessageMethod(mqm)
}

func Process(mqm *MaridQueueMessage) error {
	queuePayload := QueuePayload{}
	err := json.Unmarshal([]byte(*mqm.GetMessage().Body), &queuePayload)
	if err == nil {
		if action := queuePayload.Action; action != "" {
			payload, _ := json.Marshal(queuePayload)
			actionMappings := conf.Configuration["actionMappings"].(map[string]interface{})
			if _, ok := actionMappings[action]; ok {
				commandOutput, errorOutput, err := runbook.ExecuteRunbookMethod(action, string(payload))
				log.Println(commandOutput, errorOutput, err)
			} else {
				return errors.New("There is no mapped action found for [" + action + "]")
			}
		} else {
			return errors.New("SQS message does not contain action property")
		}
	} else {
		return errors.New(err.Error())
	}
	return nil
}

func (mqm *MaridQueueMessage) Process() error {
	return mqm.ProcessMethod(mqm)
}

func NewMaridMessage(message *sqs.Message) QueueMessage {
	mqm := &MaridQueueMessage{
		Message: message,
	}
	mqm.GetMessageMethod = GetMessage
	mqm.ProcessMethod = Process

	return mqm
}
