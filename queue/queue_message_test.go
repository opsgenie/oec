package queue

import (
	"testing"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

func TestGetMessage(t *testing.T) {
	assert := assert.New(t)

	expectedMessage := &sqs.Message{}
	expectedMessage.SetMessageId("messageId")
	expectedMessage.SetBody("messageBody")

	queueMessage := NewMaridMessage(expectedMessage)
	actualMessage := queueMessage.GetMessage()

	assert.Equal(expectedMessage, actualMessage)
}
