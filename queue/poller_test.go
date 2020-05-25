package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/oec/conf"
	"github.com/opsgenie/oec/worker_pool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

var mockPollerConf = &conf.PollerConf{
	pollingWaitIntervalInMillis,
	visibilityTimeoutInSec,
	maxNumberOfMessages,
}

func newPollerTest() *poller {
	return &poller{
		quit:        make(chan struct{}),
		wakeUp:      make(chan struct{}),
		isRunning:   false,
		isRunningWg: &sync.WaitGroup{},
		startStopMu: &sync.Mutex{},
		conf: &conf.Configuration{
			ApiKey:               mockApiKey,
			BaseUrl:              mockBaseUrl,
			PollerConf:           *mockPollerConf,
			ActionSpecifications: mockActionSpecs,
		},

		workerPool:         NewMockWorkerPool(),
		queueProvider:      NewMockQueueProvider(),
		messageHandler:     NewMockMessageHandler(),
		queueMessageLogrus: &logrus.Logger{},
	}
}

func TestStartAndStopPolling(t *testing.T) {

	poller := newPollerTest()

	err := poller.Start()
	assert.Nil(t, err)
	assert.Equal(t, true, poller.isRunning)

	err = poller.Start()
	assert.NotNil(t, err)
	assert.Equal(t, "Poller is already running.", err.Error())
	assert.Equal(t, true, poller.isRunning)

	err = poller.Stop()
	assert.Nil(t, err)
	assert.Equal(t, false, poller.isRunning)
}

func TestStopPollingNonPollingState(t *testing.T) {

	poller := newPollerTest()

	err := poller.Stop()
	assert.NotNil(t, err)
	assert.Equal(t, "Poller is not running.", err.Error())
}

func TestPollWithNoAvailableWorker(t *testing.T) {

	poller := newPollerTest()

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return 0
	}

	shouldWait := poller.poll()
	assert.True(t, shouldWait)
}

func TestPollWithReceiveError(t *testing.T) {

	poller := newPollerTest()

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return 1
	}
	poller.queueProvider.(*MockSQSProvider).ReceiveMessageFunc = func(i int64, i2 int64) ([]*sqs.Message, error) {
		return nil, errors.New("")
	}

	shouldWait := poller.poll()
	assert.True(t, shouldWait)
}

func TestPollZeroMessage(t *testing.T) {

	poller := newPollerTest()

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return 1
	}
	poller.queueProvider.(*MockSQSProvider).ReceiveMessageFunc = func(i int64, i2 int64) ([]*sqs.Message, error) {
		return []*sqs.Message{}, nil
	}

	logrus.SetLevel(logrus.DebugLevel)
	shouldWait := poller.poll()
	assert.True(t, shouldWait)
}

func TestPollMaxMessage(t *testing.T) {

	poller := newPollerTest()

	expected := 4

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return int32(expected)
	}

	maxNumberOfMessages := 0
	poller.queueProvider.(*MockSQSProvider).ReceiveMessageFunc = func(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
		maxNumberOfMessages = int(numOfMessage)
		return nil, errors.New("Receive Error")
	}

	shouldWait := poller.poll()
	assert.True(t, shouldWait)
	assert.Equal(t, expected, maxNumberOfMessages)
}

func TestPollMaxMessageUpperBound(t *testing.T) {

	poller := newPollerTest()

	availableWorkerCount := 12

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return int32(availableWorkerCount)
	}

	maxNumberOfMessages := int64(0)
	poller.queueProvider.(*MockSQSProvider).ReceiveMessageFunc = func(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
		maxNumberOfMessages = numOfMessage
		return nil, errors.New("Receive Error")
	}

	shouldWait := poller.poll()
	assert.True(t, shouldWait)
	assert.Equal(t, poller.conf.PollerConf.MaxNumberOfMessages, maxNumberOfMessages)
}

func TestPollMessageSubmitFail(t *testing.T) {

	poller := newPollerTest()

	expected := 4

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return int32(expected)
	}
	poller.queueProvider.(*MockSQSProvider).ReceiveMessageFunc = mockSuccessReceiveFunc

	submitCount := 0
	poller.workerPool.(*MockWorkerPool).SubmitFunc = func(job worker_pool.Job) (bool, error) {
		submitCount++
		return false, nil
	}

	releaseCount := 0
	poller.queueProvider.(*MockSQSProvider).ChangeMessageVisibilityFunc = func(message *sqs.Message, visibilityTimeout int64) error {
		if visibilityTimeout == 0 {
			releaseCount++
		}
		return nil
	}

	shouldWait := poller.poll()

	assert.False(t, shouldWait)
	assert.Equal(t, expected, submitCount)
	assert.Equal(t, expected, releaseCount)
}

func TestPollMessageSubmitError(t *testing.T) {

	poller := newPollerTest()

	expected := 5

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return int32(expected)
	}
	poller.queueProvider.(*MockSQSProvider).ReceiveMessageFunc = mockSuccessReceiveFunc

	submitCount := 0
	poller.workerPool.(*MockWorkerPool).SubmitFunc = func(job worker_pool.Job) (bool, error) {
		submitCount++
		return false, errors.New("Submit Error")
	}

	releaseCount := 0
	poller.queueProvider.(*MockSQSProvider).ChangeMessageVisibilityFunc = func(message *sqs.Message, visibilityTimeout int64) error {
		if visibilityTimeout == 0 {
			releaseCount++
		}
		return nil
	}

	shouldWait := poller.poll()

	assert.True(t, shouldWait)
	assert.Equal(t, 1, submitCount)
	assert.Equal(t, expected, releaseCount)
}

func TestPollMessageSubmitSuccess(t *testing.T) {

	poller := newPollerTest()

	poller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() int32 {
		return 5
	}
	poller.queueProvider.(*MockSQSProvider).ReceiveMessageFunc = mockSuccessReceiveFunc

	poller.workerPool.(*MockWorkerPool).SubmitFunc = func(job worker_pool.Job) (bool, error) {
		return true, nil
	}

	shouldWait := poller.poll()

	assert.False(t, shouldWait)
}

// Mock Poller
type MockPoller struct {
	StartPollingFunc func() error
	StopPollingFunc  func() error

	RefreshClientFunc func(assumeRoleResult AssumeRoleResult) error
	QueueProviderFunc func() SQSProvider
}

func NewMockPoller() Poller {
	return &MockPoller{}
}

func NewMockPollerForQueueProcessor(workerPool worker_pool.WorkerPool, queueProvider SQSProvider,
	messageHandler MessageHandler, conf *conf.Configuration, ownerId string) Poller {
	return NewMockPoller()
}

func (p *MockPoller) Start() error {
	if p.StartPollingFunc != nil {
		return p.StartPollingFunc()
	}
	return nil
}

func (p *MockPoller) Stop() error {
	if p.StopPollingFunc != nil {
		return p.StopPollingFunc()
	}
	return nil
}

func (p *MockPoller) RefreshClient(assumeRoleResult AssumeRoleResult) error {
	if p.RefreshClientFunc != nil {
		return p.RefreshClientFunc(assumeRoleResult)
	}
	return nil
}

func (p *MockPoller) QueueProvider() SQSProvider {
	if p.QueueProviderFunc != nil {
		return p.QueueProviderFunc()
	}
	return NewMockQueueProvider()
}
