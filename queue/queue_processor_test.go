package queue

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/opsgenie/marid2/conf"
	"net/http"
	"io"
	"github.com/pkg/errors"
)

var testQp = NewQueueProcessor().(*MaridQueueProcessor)
var defaultQp = NewQueueProcessor().(*MaridQueueProcessor)

var mockUrl1 = "https://sqs.us-west-2.amazonaws.com/255452344566/marid-test-1-2"
var mockUrl2 = "https://sqs.us-east-2.amazonaws.com/255452344566/marid-test-1-2"

var mockUrls = map[string]struct{}{
	 mockUrl1: {},
	 mockUrl2: {},
}

var mockPollers = map[Poller]struct{}{
	NewMockPoller(mockUrl1) : {},
	NewMockPoller(mockUrl2) : {},
}

func mockConvertStringListToMap(list []string) map[string]struct{} {
	return map[string]struct{}{
		mockUrl1: {},
		mockUrl2: {},
	}
}

func NewMockPoller(queueUrl string) Poller {
	return &PollerImpl{
		queueProvider: &MaridQueueProvider{queueUrl: queueUrl},
		refreshClientMethod: mockRefreshClientSuccess,
		StartPollingMethod: mockStartPollingSuccess,
		StopPollingMethod: mockStartPollingSuccess,
	}
}

func mockAddPoller(qp *MaridQueueProcessor, queueUrl *string) Poller {
	poller := NewMockPoller(*queueUrl)
	qp.pollers[poller] = struct{}{}
	return poller
}

func mockAddPollerSuccessAwsOperations(qp *MaridQueueProcessor, queueUrl *string) Poller {
	poller := addPoller(qp, queueUrl).(*PollerImpl)
	poller.changeMessageVisibility = mockChangeMessageVisibilitySuccess
	poller.receiveMessage = mockReceiveMessageSuccess
	submit := poller.submit
	poller.submit = func(job Job) (bool, error) {
		job.(*SqsJob).queueMessage.(*MaridQueueMessage).ProcessMethod = mockProcess
		job.(*SqsJob).deleteMessage = mockDeleteMessageOfPollerSuccess
		return submit(job)
	}
	return poller
}

func TestSetQueueProcessor(t *testing.T) {

	qp := NewQueueProcessor().
		setMaxNumberOfMessages(10).
		setMaxNumberOfWorker(10).
		setMinNumberOfWorker(3).
		setKeepAliveTime(time.Second).
		setMonitoringPeriod(time.Second * 5)

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
	}()

	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}
	testQp.retryer.getMethod = mockHttpGet
	testQp.addPollerMethod = mockAddPollerSuccessAwsOperations

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

	err := testQp.Stop()

	assert.NotNil(t, err)
	assert.Equal(t,"Queue processor is not running." , err.Error())
}

func TestReceiveToken(t *testing.T) {

	defer func() {
		testQp.retryer.getMethod = defaultQp.retryer.getMethod
		conf.Configuration = map[string]interface{}{}
	}()

	testQp.retryer.getMethod = mockHttpGet
	conf.Configuration = map[string]interface{}{
		"apiKey" : "",
	}

	token, err := testQp.receiveToken()

	assert.Nil(t, err)
	assert.Equal(t, "accessKeyId", token.Data.AssumeRoleResult.Credentials.AccessKeyId)
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

	url1 := "testQueueUrl1"
	poller := testQp.addPoller(&url1)
	url2 := "testQueueUrl2"
	testQp.addPoller(&url2)

	assert.Equal(t, url1, poller.GetQueueUrl())
	assert.Equal(t, testQp.pollingWaitInterval, poller.GetPollingWaitInterval())
	assert.Equal(t, testQp.maxNumberOfMessages, poller.GetMaxNumberOfMessages())
	assert.Equal(t, testQp.visibilityTimeoutInSeconds, poller.GetVisibilityTimeout())

	_, contains := testQp.pollers[poller]
	assert.True(t, contains)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshPollersRepeat(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
		convertStringListToMapMethod = convertStringListToMap
	}()

	convertStringListToMapMethod = mockConvertStringListToMap
	testQp.addPollerMethod = mockAddPoller

	testQp.refreshPollers(&mockToken)
	testQp.refreshPollers(&mockToken)
	testQp.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshPollersWithNotHavingPoller(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
		convertStringListToMapMethod = convertStringListToMap
	}()

	convertStringListToMapMethod = mockConvertStringListToMap
	testQp.addPollerMethod = mockAddPoller

	testQp.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshOldPollers(t *testing.T) {

	defer func() {
		testQp.addPollerMethod = defaultQp.addPollerMethod
		testQp.pollers = defaultQp.pollers
		convertStringListToMapMethod = convertStringListToMap
	}()

	convertStringListToMapMethod = mockConvertStringListToMap
	testQp.addPollerMethod = mockAddPoller
	testQp.pollers = mockPollers

	testQp.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(testQp.pollers))
}

func TestRefreshPollersWithEmptyAssumeRoleResult(t *testing.T) {

	testQp.refreshPollers(&mockEmptyToken)

	assert.Equal(t, 0, len(testQp.pollers))
}

func TestRefreshPollerWithEmptyToken(t *testing.T) {

	testQp.refreshPollers(&mockEmptyToken)

	assert.Equal(t, 0, len(testQp.pollers))
}





