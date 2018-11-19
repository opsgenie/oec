package queue

import (
	"math/rand"
	"time"
	"github.com/aws/aws-sdk-go/service/sqs"
	"strconv"
	"sync"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/pkg/errors"
)

var testPoller = NewPollerForTest()
var defaultPoller = NewPollerForTest()

func NewPollerForTest() *MaridPoller {
	pollingWaitInterval := time.Millisecond * 100
	maxNumberOfMessages := int64(5)
	visibilityTimeoutInSeconds := int64(15)
	return &MaridPoller{
		quit:                       make(chan struct{}),
		wakeUpChan:                 make(chan struct{}),
		state:                      INITIAL,
		startStopMutex:             &sync.Mutex{},
		pollingWaitInterval:        &pollingWaitInterval,
		maxNumberOfMessages:        &maxNumberOfMessages,
		visibilityTimeoutInSeconds: &visibilityTimeoutInSeconds,
		workerPool:                 NewMockWorkerPool(),
		queueProvider:              NewMockQueueProvider(),
		releaseMessagesMethod:      releaseMessages,
		waitMethod:                 waitPolling,
		runMethod:                  runPoller,
		wakeUpMethod:               wakeUpPoller,
		StopPollingMethod:          StopPolling,
		StartPollingMethod:         StartPolling,
		pollMethod:                 poll,
	}
}

func mockRefreshClientSuccess(assumeRoleResult *AssumeRoleResult) error {
	return nil
}

func mockStartPollingSuccess(p *MaridPoller) error {
	return nil
}

func mockStopPollingSuccess(p *MaridPoller) error {
	return nil
}

func mockChangeMessageVisibilitySuccess(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) error {
	return nil
}

func mockDeleteMessageOfPollerSuccess(mqp *MaridQueueProvider, message *sqs.Message) error {
	return nil
}

func mockReceiveMessageSuccess(mqp *MaridQueueProvider, numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	messages := make([]*sqs.Message, 0)
	for i := 1; i < rand.Intn(int(numOfMessage+1)) ; i++ {
		sqsMessage := &sqs.Message{}
		random := rand.Intn(301) * i
		sqsMessage.SetMessageId( strconv.Itoa(int(random) ))
		messages = append(messages, sqsMessage)
	}
	return messages, nil
}

func mockReceiveMessageExactNumber(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	messages := make([]*sqs.Message, 0)
	for i := int64(0); i < numOfMessage ; i++ {
		sqsMessage := &sqs.Message{}
		sqsMessage.SetMessageId(strconv.Itoa(int(i+1)))
		messages = append(messages, sqsMessage)
	}
	return messages, nil
}

func TestMultipleStartPolling(t *testing.T) {

	defer func() {
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.state = defaultPoller.state
	}()

	testPoller.runMethod = func(p *MaridPoller) {}

	err := testPoller.StartPolling()
	assert.Nil(t, err)

	err = testPoller.StartPolling()
	assert.NotNil(t, err)
	assert.Equal(t, "Poller is already running.", err.Error())
}

func TestStartPollingNonInitialState(t *testing.T) {

	defer func() {
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.state = defaultPoller.state
	}()

	testPoller.runMethod = func(p *MaridPoller) {}
	testPoller.state = POLLING

	err := testPoller.StartPolling()
	assert.NotNil(t, err)
	assert.Equal(t, "Poller is already running.", err.Error())
}

func TestStartPolling(t *testing.T) {

	defer func() {
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.state = defaultPoller.state
	}()

	testPoller.runMethod = func(p *MaridPoller) {}

	err := testPoller.StartPolling()
	assert.Nil(t, err)

	expectedState := uint32(POLLING)
	assert.Equal(t, expectedState, testPoller.state)
}

func TestStopPollingNonPollingState(t *testing.T) {

	defer func() {
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.state = defaultPoller.state
	}()

	testPoller.runMethod = func(p *MaridPoller) {}
	testPoller.state = FINISHED

	err := testPoller.StopPolling()
	assert.NotNil(t, err)
	assert.Equal(t, "Poller is not running.", err.Error())
}

func TestStopPolling(t *testing.T) {

	defer func() {
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.state = defaultPoller.state
		testPoller.quit = defaultPoller.quit
		testPoller.wakeUpChan = defaultPoller.wakeUpChan
	}()

	testPoller.runMethod = func(p *MaridPoller) {}
	testPoller.state = POLLING

	err := testPoller.StopPolling()
	assert.Nil(t, err)

	expectedState := uint32(FINISHED)
	assert.Equal(t, expectedState, testPoller.state)
}

func TestPollZeroMessage(t *testing.T) {

	defer func() {
		testPoller.workerPool = defaultPoller.workerPool
		testPoller.queueProvider = defaultPoller.queueProvider
	}()

	testPoller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() uint32 {
		return uint32(rand.Int31n(5))
	}
	testPoller.queueProvider.(*MockQueueProvider).ReceiveMessageFunc = mockReceiveZeroMessage
	testPoller.queueProvider.(*MockQueueProvider).GetMaridMetadataFunc = func() MaridMetadata {
		return mockMaridMetadata1
	}

	shouldWait := testPoller.poll()

	assert.True(t, shouldWait)
}

func TestPollWithReceiveError(t *testing.T) {

	defer func() {
		testPoller.workerPool = defaultPoller.workerPool
		testPoller.queueProvider = defaultPoller.queueProvider
	}()

	testPoller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() uint32 {
		return uint32(rand.Int31n(5))
	}
	testPoller.queueProvider.(*MockQueueProvider).ReceiveMessageFunc = mockReceiveMessageError

	shouldWait := testPoller.poll()

	assert.True(t, shouldWait)
}

func TestPollWithNoAvailableWorker(t *testing.T) {

	defer func() {
		testPoller.workerPool = defaultPoller.workerPool
	}()

	testPoller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() uint32 {
		return 0
	}

	shouldWait := testPoller.poll()

	assert.Equal(t, false, shouldWait)
}

func TestPollMessageSubmitFail(t *testing.T) {

	defer func() {
		testPoller.workerPool = defaultPoller.workerPool
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.releaseMessagesMethod = defaultPoller.releaseMessagesMethod
	}()

	expected := 5

	testPoller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() uint32 {
		return uint32(expected)
	}
	testPoller.workerPool.(*MockWorkerPool).SubmitFunc = func(job Job) (bool, error) {
		return false, nil
	}
	testPoller.queueProvider.(*MockQueueProvider).ReceiveMessageFunc = mockReceiveMessageSuccessOfProvider


	releaseCount := 0
	testPoller.releaseMessagesMethod = func(p *MaridPoller, messages []*sqs.Message) {
		releaseCount++
		return
	}

	shouldWait := testPoller.poll()

	assert.False(t, shouldWait)
	assert.Equal(t, expected, releaseCount)
}

func TestPollMessageSubmitError(t *testing.T) {

	defer func() {
		testPoller.workerPool = defaultPoller.workerPool
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.releaseMessagesMethod = defaultPoller.releaseMessagesMethod
	}()

	expected := 5

	testPoller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() uint32 {
		return uint32(expected)
	}
	testPoller.workerPool.(*MockWorkerPool).SubmitFunc = func(job Job) (bool, error) {
		return false, errors.New("Test submit error")
	}
	testPoller.queueProvider.(*MockQueueProvider).ReceiveMessageFunc = mockReceiveMessageSuccessOfProvider

	messageLength := 0
	testPoller.releaseMessagesMethod = func(p *MaridPoller, messages []*sqs.Message) {
		messageLength = len(messages)
		return
	}

	shouldWait := testPoller.poll()

	assert.True(t, shouldWait)
	assert.Equal(t, expected, messageLength)
}

func TestPollMessageSubmitSuccess(t *testing.T) {

	defer func() {
		testPoller.workerPool = defaultPoller.workerPool
		testPoller.queueProvider = defaultPoller.queueProvider
	}()

	testPoller.workerPool.(*MockWorkerPool).NumberOfAvailableWorkerFunc = func() uint32 {
		return uint32(rand.Int31n(5))
	}
	testPoller.workerPool.(*MockWorkerPool).SubmitFunc = func(job Job) (bool, error) {
		return true, nil
	}
	testPoller.queueProvider.(*MockQueueProvider).ReceiveMessageFunc = mockReceiveMessageSuccessOfProvider

	shouldWait := testPoller.poll()

	assert.False(t, shouldWait)
}

func TestReleaseMessage(t *testing.T) {

	defer func() {
		testPoller.queueProvider = defaultPoller.queueProvider
	}()

	var messageVerify []string
	testPoller.queueProvider.(*MockQueueProvider).ChangeMessageVisibilityFunc = func(message *sqs.Message, i int64) error {
		messageVerify = append(messageVerify, *message.MessageId)
		return nil
	}

	testPoller.queueProvider.(*MockQueueProvider).GetMaridMetadataFunc = func() MaridMetadata {
		return mockMaridMetadata1
	}

	messages, _ := mockReceiveMessageExactNumber(5, 15)
	testPoller.releaseMessages(messages)

	expected := []string{"1", "2", "3", "4", "5"}

	assert.Equal(t, expected, messageVerify)
}

func TestWaitPollingWakeUp(t *testing.T) {

	defer func() {
		testPoller.waitMethod = defaultPoller.waitMethod
		testPoller.state = defaultPoller.state
	}()

	isQuit := make(chan struct{})

	testPoller.waitMethod = func(p *MaridPoller, pollingWaitPeriod time.Duration) {
		waitPolling(p, pollingWaitPeriod)
		isQuit <- struct{}{}
	}

	start := time.Now()

	pollingWaitPeriod := time.Second * 5
	go testPoller.wait(pollingWaitPeriod)
	testPoller.wakeUpChan <- struct{}{}

	<- isQuit
	close(isQuit)

	took := time.Now().Sub(start)

	assert.True(t, took < pollingWaitPeriod)
}

func TestWaitPollingQuit(t *testing.T) {

	defer func() {
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.pollMethod = defaultPoller.pollMethod
		testPoller.state = defaultPoller.state
	}()

	isQuit := make(chan struct{})

	testPoller.waitMethod = func(p *MaridPoller, pollingPeriod time.Duration) {
		waitPolling(p, pollingPeriod)
		isQuit <- struct{}{}
	}

	go testPoller.wait(time.Nanosecond)

	<- isQuit
	close(isQuit)

	assert.Equal(t, uint32(POLLING), testPoller.state)
}

func TestWakeUp(t *testing.T) {

	defer func() {
		testPoller.state = defaultPoller.state
	}()

	testPoller.state = WAITING

	go testPoller.wakeUp()

	<- testPoller.wakeUpChan
}

func TestRunPollerQuit(t *testing.T) {

	defer func() {
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.pollMethod = defaultPoller.pollMethod
		testPoller.state = defaultPoller.state
	}()

	isQuit := make(chan struct{})


	testPoller.runMethod = func(p *MaridPoller) {
		runPoller(p)
		isQuit <- struct{}{}
	}

	testPoller.pollMethod = func(p *MaridPoller) (shouldWait bool) {
		return false
	}

	go testPoller.run()

	testPoller.quit <- struct{}{}
	<- isQuit
	close(isQuit)
}

func TestRunPoller(t *testing.T) {

	defer func() {
		testPoller.pollMethod = defaultPoller.pollMethod
		testPoller.waitMethod = defaultPoller.waitMethod
	}()

	waitOnce := true
	isWait := make(chan struct{})

	testPoller.pollMethod = func(p *MaridPoller) (shouldWait bool) {
		return waitOnce
	}

	testPoller.waitMethod = func(p *MaridPoller, pollingPeriod time.Duration) {
		isWait <- struct{}{}
		waitOnce = false
	}

	go testPoller.run()

	<- isWait
	close(isWait)

	testPoller.quit <- struct{}{}
}
