package queue

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"github.com/aws/aws-sdk-go/service/sqs"
	"time"
	"github.com/pkg/errors"
)

func TestGetJobId(t *testing.T) {

	expectedId := "jobId"

	sqsJob := SqsJob {
		id: &expectedId,
		GetJobIdMethod: GetJobId,
	}

	actualId := sqsJob.GetJobId()

	assert.Equal(t, expectedId, actualId)
}

func TestGetJobMessage(t *testing.T) {

	expectedMessage := &sqs.Message{}

	maridMessage := &MaridQueueMessage {
		Message: expectedMessage,
		GetMessageMethod: GetMessage,
	}

	sqsJob := SqsJob{
		queueMessage: maridMessage,
		GetMessageMethod: GetJobMessage,
	}

	actualMessage := sqsJob.GetMessage()

	assert.Equal(t, expectedMessage, actualMessage)
}

func TestSetStateToExecutingInInitialState(t *testing.T) {

	sqsJob := SqsJob {
		state: INITIAL,
		mu:	 &sync.Mutex{},
		setStateToExecutingMethod: setStateToExecuting,
	}

	wg := &sync.WaitGroup{}

	expectedCount := int32(1)
	actualCount := int32(0)

	for i := 0; i < 25 ; i++ {
		go func() {
			defer wg.Done()
			wg.Add(1)
			if sqsJob.setStateToExecuting() {
				atomic.AddInt32(&actualCount, 1)
			}
		}()
	}

	wg.Wait()
	expectedState := uint32(EXECUTING)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
	assert.Equal(t, expectedCount, actualCount)
}

func TestSetStateToExecutingInExecutingState(t *testing.T) {

	sqsJob := SqsJob {
		state: EXECUTING,
		mu:	 &sync.Mutex{},
		setStateToExecutingMethod: setStateToExecuting,
	}

	wg := &sync.WaitGroup{}

	expectedCount := int32(0)
	actualCount := int32(0)

	for i := 0; i < 25 ; i++ {
		go func() {
			defer wg.Done()
			wg.Add(1)
			if sqsJob.setStateToExecuting() {
				atomic.AddInt32(&actualCount, 1)
			}
		}()
	}

	wg.Wait()
	expectedState := uint32(EXECUTING)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
	assert.Equal(t, expectedCount, actualCount)
}

func TestExecute(t *testing.T) {

	maridMessage := &MaridQueueMessage {
		Message: &sqs.Message{},
		ProcessMethod: func(mqm *MaridQueueMessage) error {
			return nil
		},
		GetMessageMethod: GetMessage,
	}

	sqsJob := SqsJob {
		queueMessage: maridMessage,
		shouldObserve: true,
		setStateToExecutingMethod: func(j *SqsJob) bool {
			return true
		},
		observeMethod: func(j *SqsJob) {
			j.observer = time.NewTimer(time.Second)
		},
		ExecuteMethod: Execute,
		deleteMessage: func(message *sqs.Message) error {
			return nil
		},
		GetMessageMethod: GetJobMessage,
		GetJobIdMethod: func(j *SqsJob) string {
			return "jobId"
		},
	}

	err := sqsJob.Execute()

	assert.Nil(t, err)

	expectedState := uint32(FINISHED)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

func TestExecuteInNotInitialState(t *testing.T) {

	sqsJob := SqsJob {
		setStateToExecutingMethod: func(j *SqsJob) bool {
			return false
		},
		ExecuteMethod: Execute,
		GetJobIdMethod: func(j *SqsJob) string {
			return "jobId"
		},
	}

	err := sqsJob.Execute()

	assert.NotNil(t, err)
	assert.Equal(t, "Job[" + sqsJob.GetJobId() + "] is already executing.", err.Error())
}

func TestExecuteWithProcessError(t *testing.T) {

	maridMessage := &MaridQueueMessage {
		Message: &sqs.Message{},
		ProcessMethod: func(mqm *MaridQueueMessage) error {
			return errors.New("Process Error")
		},
		GetMessageMethod: GetMessage,
	}

	sqsJob := SqsJob {
		queueMessage: maridMessage,
		shouldObserve: false,
		setStateToExecutingMethod: func(j *SqsJob) bool {
			return true
		},
		ExecuteMethod: Execute,
		GetJobIdMethod: func(j *SqsJob) string {
			return "jobId"
		},
	}

	err := sqsJob.Execute()

	assert.NotNil(t, err)
	assert.Equal(t, "Process Error", err.Error())

	expectedState := uint32(ERROR)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

func TestExecuteWithDeleteError(t *testing.T) {

	maridMessage := &MaridQueueMessage {
		Message: &sqs.Message{},
		ProcessMethod: func(mqm *MaridQueueMessage) error {
			return nil
		},
		GetMessageMethod: GetMessage,
	}

	sqsJob := SqsJob {
		queueMessage: maridMessage,
		shouldObserve: false,
		setStateToExecutingMethod: func(j *SqsJob) bool {
			return true
		},
		ExecuteMethod: Execute,
		deleteMessage: func(message *sqs.Message) error {
			return errors.New("Delete Error")
		},
		GetJobIdMethod: func(j *SqsJob) string {
			return "jobId"
		},
		GetMessageMethod: GetJobMessage,
	}

	err := sqsJob.Execute()

	assert.NotNil(t, err)
	assert.Equal(t, "Delete Error", err.Error())

	expectedState := uint32(ERROR)
	actualState := sqsJob.state

	assert.Equal(t, expectedState, actualState)
}

func TestObserve(t *testing.T) {

	maridMessage := &MaridQueueMessage {
		Message: &sqs.Message{},
		GetMessageMethod: GetMessage,
	}

	sqsJob := SqsJob {
		queueMessage: maridMessage,
		state:				EXECUTING,
		exceedCount:		0,
		timeoutInSeconds:	10,
		GetJobIdMethod: func(j *SqsJob) string {
			return "jobId"
		},
		checkJobStatusMethod:	checkJobStatus,
		observeMethod:	observe,
		observePeriod:	time.Millisecond * 5,
		changeMessageVisibility: func(message *sqs.Message, visibilityTimeout int64) error {
			return nil
		},
	}

	sqsJob.observe()

	time.Sleep(time.Millisecond * 25)
	sqsJob.state = FINISHED  // or  sqsJob.observer.Stop()

	expectedExceedCount := uint32(2)
	actualExceedCount := sqsJob.exceedCount

	assert.Equal(t, expectedExceedCount, actualExceedCount)
}

/******************************************************************************************/

func TestGetQueue(t *testing.T) {

	expectedQueue := make(chan Job)

	jobQueue := &JobQueue{
		queue: expectedQueue,
	}

	actualQueue := jobQueue.GetQueue()

	assert.Equal(t, expectedQueue, actualQueue)
}

func TestIncrement(t *testing.T) {

	jobQueue := &JobQueue{
		load:	0,
	}

	wg := &sync.WaitGroup{}

	wg.Add(15)
	func() {
		for i := 0; i < 15; i++ {
			go func() {
				defer wg.Done()
				jobQueue.Increment()
			}()
		}
	}()

	wg.Wait()
	expectedLoad := uint32(15)
	actualLoad := jobQueue.load

	assert.Equal(t, expectedLoad, actualLoad)
}

func TestDecrement(t *testing.T) {

	jobQueue := &JobQueue{
		load:	15,
	}

	wg := &sync.WaitGroup{}

	wg.Add(10)
	func() {
		for i := 0; i < 10; i++ {
			go func() {
				defer wg.Done()
				jobQueue.Decrement()
			}()
		}
	}()

	wg.Wait()
	expectedLoad := uint32(5)
	actualLoad := jobQueue.load

	assert.Equal(t, expectedLoad, actualLoad)
}

func TestIsFull(t *testing.T) {

	jobQueue := &JobQueue{
		load:	15,
		queueSize:	15,
	}

	assert.True(t, jobQueue.IsFull())
}