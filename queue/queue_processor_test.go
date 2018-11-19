package queue

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/opsgenie/marid2/conf"
	"net/http"
	"io"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

var testQp = NewQueueProcessor().(*MaridQueueProcessor)
var defaultQp = NewQueueProcessor().(*MaridQueueProcessor)

var mockPollers = map[string]Poller{
	mockQueueUrl1 : NewMockPoller(NewQueueProviderForTest(mockMaridMetadata1)),
	mockQueueUrl2 : NewMockPoller(NewQueueProviderForTest(mockMaridMetadata2)),
}

func NewMockPoller(queueProvider QueueProvider) Poller {
	return &MaridPoller{
		queueProvider:      queueProvider,
		StartPollingMethod: mockStartPollingSuccess,
		StopPollingMethod:  mockStartPollingSuccess,
	}
}

func mockAddPoller(qp *MaridQueueProcessor, queueProvider QueueProvider) Poller {
	poller := NewMockPoller(queueProvider)
	qp.pollers[queueProvider.GetMaridMetadata().getQueueUrl()] = poller
	return poller
}

func mockAddPollerWithSuccessAwsOperations(qp *MaridQueueProcessor, queueProvider QueueProvider) Poller {
	poller := addPoller(qp, queueProvider).(*MaridPoller)
	poller.queueProvider.(*MaridQueueProvider).ChangeMessageVisibilityMethod = mockChangeMessageVisibilitySuccess
	poller.queueProvider.(*MaridQueueProvider).ReceiveMessageMethod = mockReceiveMessageSuccess
	poller.queueProvider.(*MaridQueueProvider).DeleteMessageMethod = mockDeleteMessageOfPollerSuccess
	submit := poller.workerPool.(*WorkerPoolImpl).submitFunc
	poller.workerPool.(*WorkerPoolImpl).submitFunc = func(wp *WorkerPoolImpl, job Job) (isSubmitted bool, err error) {
		job.(*SqsJob).queueMessage.(*MaridQueueMessage).ProcessMethod = mockProcess
		return submit(poller.workerPool.(*WorkerPoolImpl), job)
	}
	return poller
}

func TestSetQueueProcessor(t *testing.T) {

	qp := NewQueueProcessor().
		SetMaxNumberOfMessages(10).
		SetMaxNumberOfWorker(10).
		SetMinNumberOfWorker(3).
		SetKeepAliveTime(time.Second).
		SetMonitoringPeriod(time.Second * 5)

	wp := qp.(*MaridQueueProcessor).workerPool.(*WorkerPoolImpl)

	expectedMaxNumberOfMessages := uint32(10)
	actualMaxNumberOfMessages := wp.maxNumberOfWorker

	assert.Equal(t, expectedMaxNumberOfMessages, actualMaxNumberOfMessages)
}

func TestStartAndStopQueueProcessor(t *testing.T) {

	defer func() {
		conf.Configuration = map[string]interface{}{}
		testQp.retryer.getMethod = defaultQp.retryer.getMethod
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
		testQp.quit = make(chan struct{})
	}()

	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}
	testQp.retryer.getMethod = mockHttpGet
	testQp.addPollerMethod = mockAddPollerWithSuccessAwsOperations

	testQp.SetQueueSize(5).SetMaxNumberOfWorker(100)
	err := testQp.Start()
	assert.Nil(t, err)
	err = testQp.Stop()
	assert.Nil(t, err)
}

func TestStartQueueProcessorInitialError(t *testing.T) {

	defer func() {
		conf.Configuration = map[string]interface{}{}
		testQp.retryer.getMethod = defaultQp.retryer.getMethod
	}()

	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}
	testQp.retryer.getMethod = mockHttpGetError

	err := testQp.Start()

	assert.NotNil(t, err)
	assert.Equal(t,"Test http error has occurred while getting token." , err.Error())
}

func TestStopQueueProcessorWhileNotRunning(t *testing.T) {

	defer func() {
		testQp.quit = make(chan struct{})
	}()

	err := testQp.Stop()

	assert.NotNil(t, err)
	assert.Equal(t,"Queue processor is not running." , err.Error())
}

func TestReceiveToken(t *testing.T) {

	defer func() {
		testQp.pollers = defaultQp.pollers
		testQp.retryer.getMethod = defaultQp.retryer.getMethod
		conf.Configuration = map[string]interface{}{}
	}()

	testQp.pollers = mockPollers
	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}

	var actualRequest *http.Request

	testQp.retryer.getMethod = func(retryer *Retryer, request *http.Request) (*http.Response, error) {
		actualRequest = request
		return mockHttpGet(retryer, request)
	}

	token, err := testQp.receiveToken()

	assert.Nil(t, err)
	assert.Equal(t, 2, len(token.Data.MaridMetaDataList))
	assert.Equal(t, "accessKeyId1", token.Data.MaridMetaDataList[0].AssumeRoleResult.Credentials.AccessKeyId)
	assert.Equal(t, "accessKeyId2", token.Data.MaridMetaDataList[1].AssumeRoleResult.Credentials.AccessKeyId)

	for _, poller := range testQp.pollers  {
		maridMetadata := poller.GetQueueProvider().GetMaridMetadata()
		expectedQuery := maridMetadata.getRegion() + "=" + strconv.FormatInt(maridMetadata.getExpireTimeMillis(), 10)

		assert.True(t, strings.Contains(actualRequest.URL.RawQuery, expectedQuery))
	}

	//assert.Equal(t, "app.opsgenie.com", actualRequest.URL.Host)
	assert.Equal(t, "/v2/integrations/maridv2/credentials", actualRequest.URL.Path)
}

func TestReceiveTokenInvalidJson(t *testing.T) {

	defer func() {
		testQp.retryer.getMethod = defaultQp.retryer.getMethod
		conf.Configuration = map[string]interface{}{}
	}()

	testQp.retryer.getMethod = mockHttpGetInvalidJson
	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}

	_, err := testQp.receiveToken()

	assert.NotNil(t, err)
}

func TestReceiveTokenGetError(t *testing.T) {

	defer func() {
		testQp.retryer.getMethod = defaultQp.retryer.getMethod
		conf.Configuration = map[string]interface{}{}
	}()

	testQp.retryer.getMethod = mockHttpGetError
	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}

	_, err := testQp.receiveToken()

	assert.NotNil(t, err)
	assert.Equal(t, "Test http error has occurred while getting token.", err.Error())
}

func TestReceiveTokenRequestError(t *testing.T) {

	defer func() {
		httpNewRequest = http.NewRequest
	}()

	httpNewRequest = func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("Test: Http new request error.")
	}

	_, err := testQp.receiveToken()

	assert.NotNil(t, err)
	assert.Equal(t, "Test: Http new request error.", err.Error())
}

func TestReceiveTokenApiKeyError(t *testing.T) {

	_, err := testQp.receiveToken()

	assert.NotNil(t, err)
	assert.Equal(t, "The configuration does not have an api key.", err.Error())
}

func TestAddPollerTest(t *testing.T) {

	defer func() {
		testQp.pollers = defaultQp.pollers
	}()

	poller := testQp.addPoller(NewQueueProviderForTest(mockMaridMetadata1))
	testQp.addPoller(NewQueueProviderForTest(mockMaridMetadata2))

	assert.Equal(t, mockMaridMetadata1.getQueueUrl(), poller.GetQueueProvider().GetMaridMetadata().getQueueUrl())
	assert.Equal(t, testQp.pollingWaitInterval, poller.GetPollingWaitInterval())
	assert.Equal(t, testQp.maxNumberOfMessages, poller.GetMaxNumberOfMessages())
	assert.Equal(t, testQp.visibilityTimeoutInSeconds, poller.GetVisibilityTimeout())

	_, contains := testQp.pollers[mockMaridMetadata1.getQueueUrl()]
	assert.True(t, contains)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRemovePollerTest(t *testing.T) {

	defer func() {
		testQp.pollers = defaultQp.pollers
	}()

	testQp.pollers = mockPollers

	poller := testQp.removePoller(mockQueueUrl1)
	testQp.removePoller(mockQueueUrl2)

	assert.Equal(t, mockMaridMetadata1.getQueueUrl(), poller.GetQueueProvider().GetMaridMetadata().getQueueUrl())

	assert.Equal(t, 0, len(testQp.pollers))
}

func TestRefreshPollersRepeat(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
	}()

	testQp.addPollerMethod = mockAddPoller

	testQp.refreshPollers(&mockToken)
	testQp.refreshPollers(&mockToken)
	testQp.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshPollersAddAndRemove(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
	}()

	testQp.addPollerMethod = mockAddPoller

	testQp.refreshPollers(&mockToken)
	testQp.refreshPollers(&mockEmptyToken)

	assert.Equal(t, 0, len(testQp.pollers))
}

func TestRefreshPollersAdd(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
	}()

	testQp.addPollerMethod = mockAddPoller

	testQp.refreshPollers(&mockEmptyToken)
	testQp.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshPollersWithNotHavingPoller(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
	}()

	testQp.addPollerMethod = mockAddPoller

	testQp.refreshPollers(&mockToken)
	testQp.refreshPollers(&mockToken)
	testQp.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshOldPollersAlreadyHavingPollers(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
	}()

	testQp.addPollerMethod = mockAddPoller
	testQp.pollers = mockPollers

	testQp.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshPollersWithEmptyAssumeRoleResult(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
	}()

	testQp.addPollerMethod = mockAddPoller
	testQp.pollers = mockPollers

	testQp.refreshPollers(&mockTokenWithEmptyAssumeRoleResult)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshPollerWithEmptyToken(t *testing.T) {

	testQp.refreshPollers(&mockEmptyToken)

	assert.Equal(t, 0, len(testQp.pollers))
}





