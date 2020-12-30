package queue

import (
	"bytes"
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

var mockActionSpecs = conf.ActionSpecifications{
	ActionMappings: mockActionMappings,
}

var mockActionMappings = conf.ActionMappings{
	"Create": conf.MappedAction{
		SourceType: "local",
		Filepath:   "/path/to/action.bin",
		Env:        []string{"e1=v1", "e2=v2"},
		Stdout:     "/path/to/stdout",
		Stderr:     "/path/to/stderr",
	},
	"Close": conf.MappedAction{
		Type:       CustomActionType,
		SourceType: "git",
		GitOptions: git.Options{
			Url:                "testUrl",
			PrivateKeyFilepath: "testKeyPath",
			Passphrase:         "testPass",
		},
		Filepath: "oec/testConfig.json",
		Env:      []string{"e1=v1", "e2=v2"},
	},
	"Retrieve": conf.MappedAction{
		Type:       HttpActionType,
		SourceType: "local",
		Filepath:   "/path/to/executor.py",
		Env:        []string{"e1=v1", "e2=v2"},
		Stdout:     "/path/to/stdout",
		Stderr:     "/path/to/stderr",
	},
}

var mockStdout = bytes.NewBufferString("stdout")
var mockStderr = bytes.NewBufferString("stderr")

var mockActionLoggers = map[string]io.Writer{
	"/path/to/stdout": mockStdout,
	"/path/to/stderr": mockStderr,
}

func mockExecute(executablePath string, args, environmentVars []string, stdout, stderr io.Writer) error {
	return nil
}

func TestProcess(t *testing.T) {

	t.Run("TestProcessSuccessfully", testProcessSuccessfully)
	t.Run("TestProcessMappedActionNotFound", testProcessMappedActionNotFound)
	t.Run("TestProcessActionTypeNotMatched", testProcessActionTypeNotMatched)
	t.Run("TestProcessFieldMissing", testProcessFieldMissing)
	t.Run("TestProcessHttpActionSuccessfully", testProcessHttpActionSuccessfully)

	runbook.ExecuteFunc = runbook.Execute
}

func testProcessSuccessfully(t *testing.T) {

	body := `{"action":"Create", "requestId": "RequestId"}`
	id := "MessageId"
	message := sqs.Message{Body: &body, MessageId: &id}
	queueMessage := NewMessageHandler(nil, mockActionSpecs, mockActionLoggers)

	runbook.ExecuteFunc = func(executablePath string, args, environmentVars []string, stdout, stderr io.Writer) error {
		assert.Equal(t, mockStdout, stdout)
		assert.Equal(t, mockStderr, stderr)
		return nil
	}

	result, err := queueMessage.Handle(message)
	assert.Nil(t, err)
	assert.Equal(t, "Create", result.Action)
	assert.Equal(t, "RequestId", result.RequestId)
	assert.True(t, result.IsSuccessful)
}

func testProcessHttpActionSuccessfully(t *testing.T) {
	runbook.ExecuteFunc = func(executablePath string, args, environmentVars []string, stdout, stderr io.Writer) error {
		io.Copy(stdout, bytes.NewBufferString(`{"headers": {"Date": "Wed, 14 Oct 2020 08:59:30 GMT"},"body": "done", "statusCode": 200}`))
		return nil
	}

	body := `{"actionType":"http", "action":"Retrieve", "requestId": "RequestId"}`
	id := "MessageId"
	message := sqs.Message{Body: &body, MessageId: &id}
	queueMessage := NewMessageHandler(nil, mockActionSpecs, mockActionLoggers)

	result, err := queueMessage.Handle(message)
	assert.Nil(t, err)
	assert.Equal(t, "Retrieve", result.Action)
	assert.Equal(t, "RequestId", result.RequestId)
	assert.Equal(t, 200, result.HttpResponse.StatusCode)
	assert.Equal(t, map[string]string{
		"Date": "Wed, 14 Oct 2020 08:59:30 GMT",
	}, result.HttpResponse.Headers)
	assert.Equal(t, "done", result.HttpResponse.Body)
	assert.Equal(t, 200, result.HttpResponse.StatusCode)
	assert.True(t, result.IsSuccessful)
}

func testProcessMappedActionNotFound(t *testing.T) {

	runbook.ExecuteFunc = mockExecute

	body := `{"action":"Ack"}`
	message := sqs.Message{Body: &body}
	messageHandler := NewMessageHandler(nil, mockActionSpecs, mockActionLoggers)

	_, err := messageHandler.Handle(message)
	expectedErr := errors.New("There is no mapped action found for action[Ack]. SQS message with entityId[] will be ignored.")
	assert.EqualError(t, err, expectedErr.Error())
}

func testProcessActionTypeNotMatched(t *testing.T) {
	runbook.ExecuteFunc = mockExecute

	body := `{"actionType":"http", "action":"Close", "requestId": "RequestId"}`
	message := sqs.Message{Body: &body}
	messageHandler := NewMessageHandler(nil, mockActionSpecs, mockActionLoggers)

	_, err := messageHandler.Handle(message)
	expectedErr := errors.New("The mapped action found for action[Close] with type[custom] but action is coming with type[http]. " +
		"SQS message with entityId[] will be ignored.")
	assert.EqualError(t, err, expectedErr.Error())
}

func testProcessFieldMissing(t *testing.T) {

	runbook.ExecuteFunc = mockExecute

	body := `{"alert":{}}`
	message := sqs.Message{Body: &body}
	messageHandler := NewMessageHandler(nil, mockActionSpecs, mockActionLoggers)

	_, err := messageHandler.Handle(message)
	expectedErr := errors.New("SQS message with entityId[] does not contain action property.")
	assert.EqualError(t, err, expectedErr.Error())
}

// Mock Queue Message
type MockMessageHandler struct {
	HandleFunc func(message sqs.Message) (*runbook.ActionResultPayload, error)
}

func (mqm *MockMessageHandler) Handle(message sqs.Message) (*runbook.ActionResultPayload, error) {
	if mqm.HandleFunc != nil {
		return mqm.HandleFunc(message)
	}

	multip := time.Duration(rand.Int31n(100 * 3))
	time.Sleep(time.Millisecond * multip * 10) // simulate a process
	return &runbook.ActionResultPayload{}, nil
}

func NewMockMessageHandler() MessageHandler {
	return &MockMessageHandler{}
}
