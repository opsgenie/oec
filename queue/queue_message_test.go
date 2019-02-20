package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/ois/conf"
	"github.com/opsgenie/ois/git"
	"github.com/opsgenie/ois/runbook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

var (
	mockMessageId     = "mockMessageId"
	mockApiKey        = "mockApiKey"
	mockBaseUrl       = "mockBaseUrl"
	mockIntegrationId = "mockIntegrationId"
)

var mockActionMappings = &conf.ActionMappings{
	"Create": conf.MappedAction{
		SourceType:           "local",
		Filepath:             "/path/to/runbook.bin",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
	"Close": conf.MappedAction{
		SourceType: "git",
		GitOptions: git.GitOptions{
			Url:                "testUrl",
			PrivateKeyFilepath: "testKeyPath",
			Passphrase:         "testPass",
		},
		Filepath:             "ois/testConfig.json",
		EnvironmentVariables: []string{"e1=v1", "e2=v2"},
	},
}

func mockExecute(executablePath string, args []string, environmentVars []string) (s string, s2 string, e error) {
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
	queueMessage := NewOISMessage(message, mockActionMappings, nil)

	result, err := queueMessage.Process()
	assert.Nil(t, err)
	assert.Equal(t, "Create", result.Action)
}

func testProcessMappedActionNotFound(t *testing.T) {

	runbook.ExecuteFunc = mockExecute

	body := `{"action":"Ack"}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewOISMessage(message, mockActionMappings, nil)

	_, err := queueMessage.Process()
	expectedErr := errors.New("There is no mapped action found for action[Ack].")
	assert.EqualError(t, err, expectedErr.Error())
}

func testProcessFieldMissing(t *testing.T) {

	runbook.ExecuteFunc = mockExecute

	body := `{"alert":{}}`
	message := &sqs.Message{Body: &body}
	queueMessage := NewOISMessage(message, mockActionMappings, nil)

	_, err := queueMessage.Process()
	expectedErr := errors.New("SQS message does not contain action property.")
	assert.EqualError(t, err, expectedErr.Error())
}

func TestGetExePathGitNilRepositories(t *testing.T) {

	closeAction := (*mockActionMappings)["Close"]
	_, err := getExePath(&closeAction, nil)

	assert.NotNil(t, err)
	assert.EqualError(t, err, "Repositories should be provided.")
}

func TestExecuteRunbookGitNonExistingRepository(t *testing.T) {

	repositories := &git.Repositories{}

	closeAction := (*mockActionMappings)["Close"]
	_, err := getExePath(&closeAction, repositories)

	assert.NotNil(t, err)
	assert.EqualError(t, err, "Git repository[testUrl] could not be found.")
}

func TestExecuteRunbookLocal(t *testing.T) {

	createAction := (*mockActionMappings)["Create"]
	exePath, err := getExePath(&createAction, nil)

	assert.NoError(t, err)
	assert.Equal(t, createAction.Filepath, exePath)

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
