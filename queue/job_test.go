package queue

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/ois/runbook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

var mockActionResultPayload = &runbook.ActionResultPayload{Action: "MockAction"}

func newJobTest() *SqsJob {
	mockQueueMessage := NewMockQueueMessage().(*MockQueueMessage)
	mockQueueMessage.ProcessFunc = func() (payload *runbook.ActionResultPayload, e error) {
		return mockActionResultPayload, nil
	}

	return &SqsJob{
		queueProvider: NewMockQueueProvider(),
		queueMessage:  mockQueueMessage,
		executeMutex:  &sync.Mutex{},
		apiKey:        &mockApiKey,
		baseUrl:       &mockBaseUrl,
		integrationId: &mockIntegrationId,
		state:         JobInitial,
	}
}

func TestJobId(t *testing.T) {

	job := newJobTest()

	actualId := job.JobId()

	assert.Equal(t, mockMessageId, actualId)
}

func TestExecute(t *testing.T) {
	wg := &sync.WaitGroup{}

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusAccepted)

		actionResult := &runbook.ActionResultPayload{}
		body, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(body, actionResult)

		assert.Equal(t, mockActionResultPayload, actionResult)
		assert.Equal(t, "GenieKey "+mockApiKey, req.Header.Get("Authorization"))
		wg.Done()
	}))
	defer testServer.Close()

	sqsJob := newJobTest()
	sqsJob.baseUrl = &testServer.URL

	wg.Add(1)
	err := sqsJob.Execute()

	wg.Wait()
	assert.Nil(t, err)

	expectedState := int32(JobFinished)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

func TestMultipleExecute(t *testing.T) {
	wg := &sync.WaitGroup{}

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusAccepted)
		wg.Done()
	}))
	defer testServer.Close()

	sqsJob := newJobTest()
	sqsJob.baseUrl = &testServer.URL

	errorResults := make(chan error, 25)

	wg.Add(26) // 25 execute try + 1 successful execute send result to testServer
	for i := 0; i < 25; i++ {
		go func() {
			defer wg.Done()
			err := sqsJob.Execute()
			if err != nil {
				errorResults <- sqsJob.Execute()
			}
		}()
	}

	wg.Wait()
	expectedState := int32(JobFinished)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState) // only one execute finished
	assert.Equal(t, 24, len(errorResults))      // other executes will fail
}

func TestExecuteInNotInitialState(t *testing.T) {

	sqsJob := newJobTest()
	sqsJob.state = JobExecuting

	err := sqsJob.Execute()
	assert.NotNil(t, err)

	expectedErr := errors.Errorf("Job[%s] is already executing or finished.", sqsJob.JobId())
	assert.EqualError(t, err, expectedErr.Error())
}

func TestExecuteWithProcessError(t *testing.T) {

	sqsJob := newJobTest()

	sqsJob.queueMessage.(*MockQueueMessage).ProcessFunc = func() (payload *runbook.ActionResultPayload, e error) {
		return nil, errors.New("Process Error")
	}

	err := sqsJob.Execute()
	assert.NotNil(t, err)

	expectedErr := errors.Errorf("Message[%s] could not be processed: %s.", sqsJob.JobId(), "Process Error")
	assert.EqualError(t, err, expectedErr.Error())

	expectedState := int32(JobError)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

func TestExecuteWithDeleteError(t *testing.T) {

	sqsJob := newJobTest()

	sqsJob.queueProvider.(*MockQueueProvider).DeleteMessageFunc = func(message *sqs.Message) error {
		return errors.New("Delete Error")
	}

	err := sqsJob.Execute()
	assert.NotNil(t, err)

	expectedErr := errors.Errorf("Message[%s] could not be deleted from the queue[%s]: %s", sqsJob.JobId(), sqsJob.queueProvider.OISMetadata().Region(), "Delete Error")
	assert.EqualError(t, err, expectedErr.Error())

	expectedState := int32(JobError)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

func TestExecuteWithInvalidQueueMessage(t *testing.T) {

	sqsJob := newJobTest()

	sqsJob.queueMessage.(*MockQueueMessage).MessageFunc = func() *sqs.Message {
		falseIntegrationId := "falseIntegrationId"
		messageAttr := map[string]*sqs.MessageAttributeValue{integrationId: {StringValue: &falseIntegrationId}}
		return &sqs.Message{MessageAttributes: messageAttr, MessageId: &mockMessageId}
	}

	err := sqsJob.Execute()
	assert.NotNil(t, err)

	expectedErr := errors.Errorf("Message[%s] is invalid, will not be processed.", sqsJob.JobId())
	assert.EqualError(t, err, expectedErr.Error())

	expectedState := int32(JobError)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

// Mock Job
type MockJob struct {
	JobIdFunc   func() string
	ExecuteFunc func() error
}

func NewMockJob() *MockJob {
	return &MockJob{}
}

func (mj *MockJob) JobId() string {
	if mj.JobIdFunc != nil {
		return mj.JobIdFunc()
	}
	return "mockJobId"
}

func (mj *MockJob) Execute() error {
	if mj.ExecuteFunc != nil {
		return mj.ExecuteFunc()
	}
	return nil
}
