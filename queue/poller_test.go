package queue

import (
	"testing"
	"math/rand"
	"github.com/stretchr/testify/assert"
	"time"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"strconv"
	"sync"
)

var testPoller = NewPollerForTest().(*PollerImpl)
var defaultPoller = NewPollerForTest().(*PollerImpl)

func NewPollerForTest() Poller {
	pollingWaitInterval := time.Millisecond * 100
	maxNumberOfMessages := int64(5)
	visibilityTimeoutInSeconds := int64(15)
	return &PollerImpl{
		quit:                       make(chan struct{}),
		wakeUpChan:                 make(chan struct{}),
		state:                      INITIAL,
		startStopMu:                &sync.Mutex{},
		pollingWaitInterval:        &pollingWaitInterval,
		maxNumberOfMessages:        &maxNumberOfMessages,
		visibilityTimeoutInSeconds: &visibilityTimeoutInSeconds,
		queueProvider:              NewQueueProviderForTest(mockMaridMetadata1),
		releaseMessagesMethod:      releaseMessages,
		waitMethod:              	waitPolling,
		runMethod:               	runPoller,
		wakeUpMethod:            	wakeUpPoller,
		StopPollingMethod:       	StopPolling,
		StartPollingMethod:      	StartPolling,
		pollMethod:              	poll,
	}
}

func mockRefreshClientSuccess(assumeRoleResult *AssumeRoleResult) error {
	return nil
}

func mockStartPollingSuccess(p *PollerImpl) error {
	return nil
}

func mockStopPollingSuccess(p *PollerImpl) error {
	return nil
}

func mockChangeMessageVisibilitySuccess(message *sqs.Message, visibilityTimeout int64) error {
	return nil
}

func mockDeleteMessageOfPollerSuccess(message *sqs.Message) error {
	return nil
}

func mockReceiveMessageSuccess(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	messages := make([]*sqs.Message, 0)
	for i := int32(0); i < rand.Int31n(int32(numOfMessage)) ; i++ {
		sqsMessage := &sqs.Message{}
		sqsMessage.SetMessageId(strconv.Itoa(int(i+1)))
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

	testPoller.runMethod = func(p *PollerImpl) {}

	err := testPoller.StartPolling()
	assert.Nil(t, err)

	err = testPoller.StartPolling()
	assert.NotNil(t, err)
	assert.Equal(t, "Poller is already running.", err.Error())
}

func TestStartPollingNonInitialState(t *testing.T) {

	defer func() {
		testPoller.getNumberOfAvailableWorker = defaultPoller.getNumberOfAvailableWorker
		testPoller.runMethod = defaultPoller.runMethod
		testPoller.state = defaultPoller.state
	}()

	testPoller.runMethod = func(p *PollerImpl) {}
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

	testPoller.runMethod = func(p *PollerImpl) {}

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

	testPoller.runMethod = func(p *PollerImpl) {}
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

	testPoller.runMethod = func(p *PollerImpl) {}
	testPoller.state = POLLING

	err := testPoller.StopPolling()
	assert.Nil(t, err)

	expectedState := uint32(FINISHED)
	assert.Equal(t, expectedState, testPoller.state)
}

func TestPollZeroMessage(t *testing.T) {

	defer func() {
		testPoller.getNumberOfAvailableWorker = defaultPoller.getNumberOfAvailableWorker
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.receiveMessage = defaultPoller.receiveMessage
	}()

	testPoller.getNumberOfAvailableWorker = func() uint32 {
		return uint32(rand.Int31n(5))
	}
	testPoller.queueProvider.(*MaridQueueProvider).ReceiveMessageMethod = mockReceiveZeroMessage
	testPoller.receiveMessage = testPoller.queueProvider.ReceiveMessage

	shouldWait := testPoller.poll()

	assert.True(t, shouldWait)
}

func TestPollWithReceiveError(t *testing.T) {

	defer func() {
		testPoller.getNumberOfAvailableWorker = defaultPoller.getNumberOfAvailableWorker
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.receiveMessage = defaultPoller.receiveMessage
	}()

	testPoller.getNumberOfAvailableWorker = func() uint32 {
		return uint32(rand.Int31n(5))
	}
	testPoller.queueProvider.(*MaridQueueProvider).ReceiveMessageMethod = mockReceiveMessageError
	testPoller.receiveMessage = testPoller.queueProvider.ReceiveMessage

	shouldWait := testPoller.poll()

	assert.True(t, shouldWait)
}

func TestPollWithNoAvailableWorker(t *testing.T) {

	defer func() {
		testPoller.getNumberOfAvailableWorker = defaultPoller.getNumberOfAvailableWorker
	}()

	testPoller.getNumberOfAvailableWorker = func() uint32 {
		return 0
	}

	shouldWait := testPoller.poll()

	assert.Equal(t, false, shouldWait)
}

func TestPollMessageSubmitFail(t *testing.T) {

	defer func() {
		testPoller.getNumberOfAvailableWorker = defaultPoller.getNumberOfAvailableWorker
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.receiveMessage = defaultPoller.receiveMessage
		testPoller.submit = defaultPoller.submit
		testPoller.releaseMessagesMethod = defaultPoller.releaseMessagesMethod
	}()

	expected := 5

	testPoller.getNumberOfAvailableWorker = func() uint32 {
		return uint32(expected)
	}
	testPoller.queueProvider.(*MaridQueueProvider).ReceiveMessageMethod = mockReceiveMessageSuccessOfProvider
	testPoller.receiveMessage = testPoller.queueProvider.ReceiveMessage
	testPoller.submit = func(job Job) (bool, error) {
		return false, nil
	}

	releaseCount := 0
	testPoller.releaseMessagesMethod = func(p *PollerImpl, messages []*sqs.Message) {
		releaseCount++
		return
	}

	shouldWait := testPoller.poll()

	assert.False(t, shouldWait)
	assert.Equal(t, expected, releaseCount)
}

func TestPollMessageSubmitError(t *testing.T) {

	defer func() {
		testPoller.getNumberOfAvailableWorker = defaultPoller.getNumberOfAvailableWorker
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.receiveMessage = defaultPoller.receiveMessage
		testPoller.submit = defaultPoller.submit
		testPoller.releaseMessagesMethod = defaultPoller.releaseMessagesMethod
	}()

	expected := 5

	testPoller.getNumberOfAvailableWorker = func() uint32 {
		return uint32(expected)
	}
	testPoller.queueProvider.(*MaridQueueProvider).ReceiveMessageMethod = mockReceiveMessageSuccessOfProvider
	testPoller.receiveMessage = testPoller.queueProvider.ReceiveMessage
	testPoller.submit = func(job Job) (bool, error) {
		return false, errors.New("Test submit error")
	}

	messageLength := 0
	testPoller.releaseMessagesMethod = func(p *PollerImpl, messages []*sqs.Message) {
		messageLength = len(messages)
		return
	}

	shouldWait := testPoller.poll()

	assert.True(t, shouldWait)
	assert.Equal(t, expected, messageLength)
}

func TestPollMessageSubmitSuccess(t *testing.T) {

	defer func() {
		testPoller.getNumberOfAvailableWorker = defaultPoller.getNumberOfAvailableWorker
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.receiveMessage = defaultPoller.receiveMessage
		testPoller.submit = defaultPoller.submit
	}()

	testPoller.getNumberOfAvailableWorker = func() uint32 {
		return uint32(rand.Int31n(5))
	}
	testPoller.queueProvider.(*MaridQueueProvider).ReceiveMessageMethod = mockReceiveMessageSuccessOfProvider
	testPoller.receiveMessage = testPoller.queueProvider.ReceiveMessage
	testPoller.submit = func(job Job) (bool, error) {
		return true, nil
	}

	shouldWait := testPoller.poll()

	assert.False(t, shouldWait)
}

func TestReleaseMessage(t *testing.T) {

	defer func() {
		testPoller.queueProvider = defaultPoller.queueProvider
		testPoller.changeMessageVisibility = defaultPoller.changeMessageVisibility
	}()

	var messageVerify []string
	testPoller.queueProvider.(*MaridQueueProvider).ChangeMessageVisibilityMethod = func(mqp *MaridQueueProvider, message *sqs.Message, visibilityTimeout int64) error {
		messageVerify = append(messageVerify, *message.MessageId)
		return nil
	}
	testPoller.changeMessageVisibility = testPoller.queueProvider.ChangeMessageVisibility

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

	testPoller.waitMethod = func(p *PollerImpl, pollingWaitPeriod time.Duration) {
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

	testPoller.waitMethod = func(p *PollerImpl, pollingPeriod time.Duration) {
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


	testPoller.runMethod = func(p *PollerImpl) {
		runPoller(p)
		isQuit <- struct{}{}
	}

	testPoller.pollMethod = func(p *PollerImpl) (shouldWait bool) {
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

	testPoller.pollMethod = func(p *PollerImpl) (shouldWait bool) {
		return waitOnce
	}

	testPoller.waitMethod = func(p *PollerImpl, pollingPeriod time.Duration) {
		isWait <- struct{}{}
		waitOnce = false
	}

	go testPoller.run()

	<- isWait
	close(isWait)

	testPoller.quit <- struct{}{}
}
