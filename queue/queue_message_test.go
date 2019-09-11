package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/oec/conf"
	"github.com/opsgenie/oec/git"
	"github.com/opsgenie/oec/runbook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"math/rand"
	"testing"
	"time"
)

var (
	mockMessageId = "mockMessageId"
	mockApiKey    = "mockApiKey"
	mockBaseUrl   = "mockBaseUrl"
	mockOwnerId   = "mockOwnerId"
)

var mockActionSpecs = &conf.ActionSpecifications{
	ActionMappings: mockActionMappings,
}

var mockActionMappings = conf.ActionMappings{
	"Create": conf.MappedAction{
		SourceType: "local",
		Filepath:   "/path/to/runbook.bin",
		Env:        []string{"e1=v1", "e2=v2"},
	},
	"Close": conf.MappedAction{
		SourceType: "git",
		GitOptions: git.GitOptions{
			Url:                "testUrl",
			PrivateKeyFilepath: "testKeyPath",
			Passphrase:         "testPass",
		},
		Filepath: "oec/testConfig.json",
		Env:      []string{"e1=v1", "e2=v2"},
	},
}

func mockExecute(executablePath string, args, environmentVars []string, stdout, stderr io.Writer) error {
	return nil
}

func TestGetMessage(t *testing.T) {

	expectedMessage := &sqs.Message{}
	expectedMessage.SetMessageId("messageId")
	expectedMessage.SetBody("messageBody")

	queueMessage := NewOECMessage(expectedMessage, nil, mockActionSpecs)
	actualMessage := queueMessage.Message()

	assert.Equal(t, expectedMessage, actualMessage)
}

func TestProcess(t *testing.T) {

	t.Run("TestProcessSuccessfully", testProcessSuccessfully)
	t.Run("TestProcessMappedActionNotFound", testProcessMappedActionNotFound)
	t.Run("TestProcessFieldMissing", testProcessFieldMissing)

	runbook.ExecuteFunc = runbook.Execute
}

func testProcessSuccessfully(t *testing.T) {

	runbook.ExecuteFunc = mockExecute

	body := `{"action":"Create"}`
	id := "MessageId"
	message := &sqs.Message{Body: &body, MessageId: &id}
	queueMessage := NewOECMessage(message, nil, mockActionSpecs)

	result, err := queueMessage.Process()
	assert.Nil(t, err)
	assert.Equal(t, "Create", result.Action)
}

func testProcessMappedActionNotFound(t *testing.T) {

	runbook.ExecuteFunc = mockExecute

	body := `{"action":"Ack"}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewOECMessage(message, nil, mockActionSpecs)

	_, err := queueMessage.Process()
	expectedErr := errors.New("There is no mapped action found for action[Ack]. SQS message with entityId[] will be ignored.")
	assert.EqualError(t, err, expectedErr.Error())
}

func testProcessFieldMissing(t *testing.T) {

	runbook.ExecuteFunc = mockExecute

	body := `{"alert":{}}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewOECMessage(message, nil, mockActionSpecs)

	_, err := queueMessage.Process()
	expectedErr := errors.New("SQS message with entityId[] does not contain action property.")
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
	messageAttr := map[string]*sqs.MessageAttributeValue{ownerId: {StringValue: &mockOwnerId}}

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
