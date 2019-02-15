package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/git"
	"github.com/opsgenie/marid2/runbook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

var mockActionMappings = &conf.ActionMappings{
	"action1": conf.MappedAction{SourceType: "local"},
	"action2": conf.MappedAction{SourceType: "github"},
}

var (
	mockMessageId     = "mockMessageId"
	mockApiKey        = "mockApiKey"
	mockBasePath      = "mockBasePath"
	mockBaseUrl       = "mockBaseUrl"
	mockIntegrationId = "mockIntegrationId"
)

func mockExecuteRunbook(mappedAction *conf.MappedAction, repositories *git.Repositories, arg []string) (string, string, error) {
	return "Operation executed successfully!", "", nil
}

func TestGetMessage(t *testing.T) {

	expectedMessage := &sqs.Message{}
	expectedMessage.SetMessageId("messageId")
	expectedMessage.SetBody("messageBody")

	queueMessage := NewOISMessage(expectedMessage, mockActionMappings, nil)
	actualMessage := queueMessage.Message()

	assert.Equal(t, expectedMessage, actualMessage)
}

func TestProcessSuccessfully(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	body := `{"action":"action1"}`
	id := "MessageId"
	message := &sqs.Message{Body: &body, MessageId: &id}
	queueMessage := NewOISMessage(message, mockActionMappings, nil)

	result, err := queueMessage.Process()
	assert.Nil(t, err)
	assert.Equal(t, "action1", result.Action)
}

func TestProcessMappedActionNotFound(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	body := `{"action":"action3"}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewOISMessage(message, mockActionMappings, nil)

	_, err := queueMessage.Process()
	expectedErr := errors.New("There is no mapped action found for [action3]")
	assert.EqualError(t, err, expectedErr.Error())
}

func TestProcessFieldMissing(t *testing.T) {

	oldExecuteRunbook := runbook.ExecuteRunbookFunc
	defer func() { runbook.ExecuteRunbookFunc = oldExecuteRunbook }()
	runbook.ExecuteRunbookFunc = mockExecuteRunbook

	body := `{"alert":{}}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewOISMessage(message, mockActionMappings, nil)

	_, err := queueMessage.Process()
	expectedErr := errors.New("SQS message does not contain action property")
	assert.EqualError(t, err, expectedErr.Error())
}

// Mock Queue Message
type MockQueueMessage struct {
	MessageFunc func() *sqs.Message
	ProcessFunc func() (*runbook.ActionResultPayload, error)
}

func (mqm *MockQueueMessage) Message() *sqs.Message {
	if mqm.MessageFunc != nil {
		return mqm.MessageFunc()
	}

	body := "mockBody"
	messageAttr := map[string]*sqs.MessageAttributeValue{integrationId: {StringValue: &mockIntegrationId}}

	return &sqs.Message{
		MessageId:         &mockMessageId,
		Body:              &body,
		MessageAttributes: messageAttr,
	}
}

func (mqm *MockQueueMessage) Process() (*runbook.ActionResultPayload, error) {
	if mqm.ProcessFunc != nil {
		return mqm.ProcessFunc()
	}

	multip := time.Duration(rand.Int31n(100 * 3))
	time.Sleep(time.Millisecond * multip * 10) // simulate a process
	return &runbook.ActionResultPayload{}, nil
}

func NewMockQueueMessage() QueueMessage {
	return &MockQueueMessage{}
}
