package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func newJobTest() *SqsJob {
	queueMessage := NewMockQueueMessage().(*MockQueueMessage)
	queueMessage.ProcessFunc = func() error {
		return nil
	}

	return &SqsJob {
		queueProvider:          NewMockQueueProvider(),
		queueMessage:           queueMessage,
		id:                     queueMessage.Message().MessageId,
		executeMutex:           &sync.Mutex{},
		state:                  JobInitial,
	}
}

func TestJobId(t *testing.T) {

	job := newJobTest()

	actualId := job.JobId()

	assert.Equal(t, "mockMessageId", actualId)
}

func TestExecute(t *testing.T) {

	sqsJob := newJobTest()

	err := sqsJob.Execute()
	assert.Nil(t, err)

	expectedState := int32(JobFinished)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

func TestMultipleExecute(t *testing.T) {

	sqsJob := newJobTest()

	wg := &sync.WaitGroup{}

	errorResults := make(chan error, 25)

	for i := 0; i < 25 ; i++ {
		go func() {
			defer wg.Done()
			wg.Add(1)
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
	assert.Equal(t, "Job[" + sqsJob.JobId() + "] is already executing.", err.Error())
}

func TestExecuteWithProcessError(t *testing.T) {

	sqsJob := newJobTest()

	sqsJob.queueMessage.(*MockQueueMessage).ProcessFunc = func() error {
		return errors.New("Process Error")
	}

	err := sqsJob.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "Process Error", err.Error())

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
	assert.Equal(t, "Delete Error", err.Error())

	expectedState := int32(JobError)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

// Mock Job
type MockJob struct {

	JobIdFunc func() string
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