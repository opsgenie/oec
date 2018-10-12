package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"time"
	"math/rand"
)

type QueueMessage interface {
	GetMessage() *sqs.Message
	Process() error
}

type MaridQueueMessage struct {
	*sqs.Message

	GetMessageMethod func(mqm *MaridQueueMessage) *sqs.Message
	ProcessMethod func(mqm *MaridQueueMessage) error
}

func GetMessage(mqm *MaridQueueMessage) *sqs.Message {
	return mqm.Message
}

func (mqm *MaridQueueMessage) GetMessage() *sqs.Message {
	return mqm.GetMessageMethod(mqm)
}

func Process(mqm *MaridQueueMessage) error {
	multip := time.Duration(rand.Int31n(100 * 3))
	time.Sleep(time.Millisecond * multip * 10)	// simulate a process
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