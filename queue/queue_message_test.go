package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/runbook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
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

func TestProcessSuccessfully(t *testing.T) {
	oldParseJsonMethod := conf.ParseJson
	defer func() { conf.ParseJson = oldParseJsonMethod }()
	conf.ParseJson = mockParseJson

	conf.Configuration = make(map[string]interface{})
	conf.Configuration["doAction"] = "mappedAction"

	oldExecuteRunbook := runbook.ExecuteRunbookMethod
	defer func() { runbook.ExecuteRunbookMethod = oldExecuteRunbook }()
	runbook.ExecuteRunbookMethod = mockExecuteRunbook

	body := "{'action' : 'doAction'}"
	queueMessage := NewMaridMessage(&sqs.Message{
		Body: &body,
	})
	err := queueMessage.Process()
	assert.Nil(t, err)
}

func TestProcessMappedActionNotFound(t *testing.T) {
	oldParseJsonMethod := conf.ParseJson
	defer func() { conf.ParseJson = oldParseJsonMethod }()
	conf.ParseJson = mockParseJson

	conf.Configuration = make(map[string]interface{})
	conf.Configuration["doNotAction"] = "mappedAction"

	oldExecuteRunbook := runbook.ExecuteRunbookMethod
	defer func() { runbook.ExecuteRunbookMethod = oldExecuteRunbook }()
	runbook.ExecuteRunbookMethod = mockExecuteRunbook
	body := "{'action' : 'doAction'}"
	queueMessage := NewMaridMessage(&sqs.Message{
		Body: &body,
	})
	err := queueMessage.Process()
	expected := errors.New("There is no mapped action found for [doAction]").Error()
	assert.Equal(t, expected, err.Error())
}

func TestProcessJSONFieldMissing(t *testing.T) {
	oldParseJsonMethod := conf.ParseJson
	defer func() { conf.ParseJson = oldParseJsonMethod }()
	conf.ParseJson = mockParseJsonEmpty

	conf.Configuration = make(map[string]interface{})
	conf.Configuration["doAction"] = "mappedAction"

	oldExecuteRunbook := runbook.ExecuteRunbookMethod
	defer func() { runbook.ExecuteRunbookMethod = oldExecuteRunbook }()
	runbook.ExecuteRunbookMethod = mockExecuteRunbook
	body := "{'action' : 'doAction'}"
	queueMessage := NewMaridMessage(&sqs.Message{
		Body: &body,
	})
	err := queueMessage.Process()
	expected := errors.New("SQS message does not contain action property").Error()
	assert.Equal(t, expected, err.Error())
}

func TestJSONError(t *testing.T) {
	oldParseJsonMethod := conf.ParseJson
	defer func() { conf.ParseJson = oldParseJsonMethod }()
	conf.ParseJson = mockParseJsonWrong

	conf.Configuration = make(map[string]interface{})
	conf.Configuration["doAction"] = "mappedAction"

	oldExecuteRunbook := runbook.ExecuteRunbookMethod
	defer func() { runbook.ExecuteRunbookMethod = oldExecuteRunbook }()
	runbook.ExecuteRunbookMethod = mockExecuteRunbook
	body := "{'action' : 'doAction'}"
	queueMessage := NewMaridMessage(&sqs.Message{
		Body: &body,
	})
	err := queueMessage.Process()
	expected := errors.New("JSON not parsed").Error()
	assert.Equal(t, expected, err.Error())
}

func mockParseJson(content []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["action"] = "doAction"
	return result, nil
}

func mockParseJsonEmpty(content []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	return result, nil
}

func mockParseJsonWrong(content []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	return result, errors.New("JSON not parsed")
}

func mockExecuteRunbook(action string) (string, string, error) {
	return "Operation executed successfully!", "(with no errors!)", nil
}
