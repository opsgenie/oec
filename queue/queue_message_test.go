package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/runbook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"math/rand"
)

var mockActionMappings = &conf.ActionMappings{
	"action1": conf.MappedAction{Source: "local"},
	"action2": conf.MappedAction{Source: "github"},
}

var mockApiKey = "mockApiKey"

func mockParseJson(content []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["action"] = "doAction"
	return result, nil
}

func mockExecuteRunbook(mappedAction *conf.MappedAction, arg string) (string, string, error) {
	return "Operation executed successfully!", "(with no errors!)", nil
}

func TestGetMessage(t *testing.T) {

	expectedMessage := &sqs.Message{}
	expectedMessage.SetMessageId("messageId")
	expectedMessage.SetBody("messageBody")

	queueMessage := NewMaridMessage(expectedMessage, mockActionMappings, &mockApiKey)
	actualMessage := queueMessage.Message()

	assert.Equal(t, expectedMessage, actualMessage)
}

func TestProcessSuccessfully(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	body := `{"action":"action1"}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewMaridMessage(message, mockActionMappings, &mockApiKey)

	err := queueMessage.Process()
	assert.Nil(t, err)
}

func TestProcessMappedActionNotFound(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	body := `{"action":"action3"}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewMaridMessage(message, mockActionMappings, &mockApiKey)

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
	queueMessage := NewMaridMessage(message, mockActionMappings, &mockApiKey)

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
