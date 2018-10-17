package queue

import (
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
	payload, err := conf.ParseJson([]byte(*mqm.GetMessage().Body))
	if err == nil {
		if action, ok := payload["action"].(string); ok {
			if mappedAction, ok := conf.Configuration[action].(string); ok {
				commandOutput, errorOutput, err := runbook.ExecuteRunbookMethod(mappedAction)
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
