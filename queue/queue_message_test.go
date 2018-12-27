package queue

import (
	"bytes"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/runbook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"
)

var mockActionMappings = &conf.ActionMappings{
	"action1": conf.MappedAction{Source: "local"},
	"action2": conf.MappedAction{Source: "github"},
}

var mockApiKey = "mockApiKey"
var mockBasePath = "mockBasePath"
var mockBaseUrl = "mockBaseUrl"

func mockExecuteRunbook(mappedAction *conf.MappedAction, arg string) (string, string, error) {
	return "Operation executed successfully!", "(with no errors!)", nil
}

func TestGetMessage(t *testing.T) {

	expectedMessage := &sqs.Message{}
	expectedMessage.SetMessageId("messageId")
	expectedMessage.SetBody("messageBody")

	queueMessage := NewMaridMessage(expectedMessage, mockActionMappings, &mockApiKey, &mockBasePath)
	actualMessage := queueMessage.Message()

	assert.Equal(t, expectedMessage, actualMessage)
}

func TestProcessSuccessfully(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	oldSendResultToOpsgenie := runbook.SendResultToOpsGenie
	defer func() { runbook.SendResultToOpsGenieFunc = oldSendResultToOpsgenie }()
	runbook.SendResultToOpsGenieFunc = func(resultPayload *runbook.ActionResultPayload, apiKey *string, baseUrl *string) error {
		return nil
	}

	body := `{"action":"action1"}`
	id := "MessageId"
	message := &sqs.Message{Body: &body, MessageId: &id}
	queueMessage := NewMaridMessage(message, mockActionMappings, &mockApiKey, &mockBasePath)

	defer func() {
		logrus.SetOutput(ioutil.Discard)
	}()

	buffer := &bytes.Buffer{}
	logrus.SetOutput(buffer)
	logrus.SetLevel(logrus.DebugLevel)

	err := queueMessage.Process()
	assert.Nil(t, err)
	assert.Contains(t, buffer.String(), "Successfully sent result to OpsGenie.")
}

func TestProcessMappedActionNotFound(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	body := `{"action":"action3"}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewMaridMessage(message, mockActionMappings, &mockApiKey, &mockBasePath)

	err := queueMessage.Process()
	expected := errors.New("There is no mapped action found for [action3]").Error()
	assert.Equal(t, expected, err.Error())
}

func TestProcessFieldMissing(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	body := `{"alert":{}}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewMaridMessage(message, mockActionMappings, &mockApiKey, &mockBasePath)

	err := queueMessage.Process()
	expected := errors.New("SQS message does not contain action property").Error()
	assert.Equal(t, expected, err.Error())
}

// Mock Queue Message
type MockQueueMessage struct {
	MessageFunc func() *sqs.Message
	ProcessFunc func() error
}

func (mqm *MockQueueMessage) Message() *sqs.Message {
	if mqm.MessageFunc != nil {
		return mqm.MessageFunc()
	}
	messageId := "mockMessageId"
	body := "mockBody"

	return &sqs.Message{
		MessageId: 	&messageId,
		Body:		&body,
	}
}

func (mqm *MockQueueMessage) Process() error {
	if mqm.ProcessFunc != nil {
		return mqm.ProcessFunc()
	}

	multip := time.Duration(rand.Int31n(100 * 3))
	time.Sleep(time.Millisecond * multip * 10)	// simulate a process
	return nil
}

func NewMockQueueMessage() QueueMessage {
	return &MockQueueMessage{}
}
