package queue

import (
	"sync"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/opsgenie/marid2/conf"
	"net/http"
	"strconv"
	"strings"
	"io"
	"github.com/pkg/errors"
	"time"
	"github.com/opsgenie/marid2/retryer"
	"encoding/json"
	"bytes"
	"io/ioutil"
)

var mockConf = &conf.Configuration{
	ApiKey:		"ApiKey",
	PollerConf: *mockPollerConf,
	PoolConf: 	*mockPoolConf,
}

func newQueueProcessorTest() *MaridQueueProcessor {

	return &MaridQueueProcessor{
		successRefreshPeriod: 	successRefreshPeriod,
		errorRefreshPeriod:   	errorRefreshPeriod,
		workerPool:           	NewMockWorkerPool(),
		conf:           		mockConf,
		pollers:              	make(map[string]Poller),
		quit:                 	make(chan struct{}),
		isRunning:            	false,
		isRunningWaitGroup:   	&sync.WaitGroup{},
		startStopMutex:       	&sync.Mutex{},
		retryer:              	&retryer.Retryer{},
	}
}

var mockPollers = map[string]Poller{
	mockQueueUrl1 : NewMockPoller(),
	mockQueueUrl2 : NewMockPoller(),
}

func mockHttpGetError(retryer *retryer.Retryer, request *http.Request) (*http.Response, error) {
	return nil, errors.New("Test http error has occurred while getting token.")
}

func mockHttpGet(retryer *retryer.Retryer, request *http.Request) (*http.Response, error) {

	token, _ := json.Marshal(mockToken)
	buff := bytes.NewBuffer(token)

	response := &http.Response{}
	response.StatusCode = 200
	response.Body = ioutil.NopCloser(buff)

	return response, nil
}

func mockHttpGetInvalidJson(retryer *retryer.Retryer, request *http.Request) (*http.Response, error) {

	response := &http.Response{}
	response.Body = ioutil.NopCloser( bytes.NewBufferString(`{"Invalid json": }`))

	return response, nil
}

func TestValidateNewQueueProcessor(t *testing.T) {
	configuration := &conf.Configuration{}
	processor := NewQueueProcessor(configuration).(*MaridQueueProcessor)

	assert.Equal(t, int64(maxNumberOfMessages), processor.conf.PollerConf.MaxNumberOfMessages)
	assert.Equal(t, int64(visibilityTimeoutInSec), processor.conf.PollerConf.VisibilityTimeoutInSeconds)
	assert.Equal(t, time.Duration(pollingWaitIntervalInMillis), processor.conf.PollerConf.PollingWaitIntervalInMillis)
}

func TestStartAndStopQueueProcessor(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	processor.retryer.DoFunc = mockHttpGet
	newPoller = NewMockPollerForQueueProcessor

	err := processor.StartProcessing()
	assert.Nil(t, err)

	assert.Equal(t, 2, len(processor.pollers))

	err = processor.StopProcessing()
	assert.Nil(t, err)
}

func TestStartQueueProcessorAndRefresh(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	processor.retryer.DoFunc = mockHttpGet
	processor.successRefreshPeriod = time.Nanosecond
	newPoller = NewMockPollerForQueueProcessor

	err := processor.StartProcessing()
	assert.Nil(t, err)

	time.Sleep(time.Nanosecond * 100)

	assert.Equal(t, 2, len(processor.pollers))
	assert.Equal(t, successRefreshPeriod, processor.successRefreshPeriod)

	err = processor.StopProcessing()
	assert.Nil(t, err)
}

func TestStartQueueProcessorInitialError(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	processor.retryer.DoFunc = mockHttpGetError
	newPoller = NewMockPollerForQueueProcessor

	err := processor.StartProcessing()

	assert.NotNil(t, err)
	assert.Equal(t,"Test http error has occurred while getting token." , err.Error())
}

func TestStopQueueProcessorWhileNotRunning(t *testing.T) {

	processor := newQueueProcessorTest()

	err := processor.StopProcessing()

	assert.NotNil(t, err)
	assert.Equal(t,"Queue processor is not running." , err.Error())
}

func TestReceiveToken(t *testing.T) {

	processor := newQueueProcessorTest()

	processor.pollers = mockPollers

	var actualRequest *http.Request

	processor.retryer.DoFunc = func(retryer *retryer.Retryer, request *http.Request) (*http.Response, error) {
		actualRequest = request
		return mockHttpGet(retryer, request)
	}

	token, err := processor.receiveToken()

	assert.Nil(t, err)
	assert.Equal(t, 2, len(token.Data.MaridMetaDataList))
	assert.Equal(t, "accessKeyId1", token.Data.MaridMetaDataList[0].AssumeRoleResult.Credentials.AccessKeyId)
	assert.Equal(t, "accessKeyId2", token.Data.MaridMetaDataList[1].AssumeRoleResult.Credentials.AccessKeyId)

	for _, poller := range processor.pollers  {
		maridMetadata := poller.QueueProvider().MaridMetadata()
		expectedQuery := maridMetadata.getRegion() + "=" + strconv.FormatInt(maridMetadata.getExpireTimeMillis(), 10)

		assert.True(t, strings.Contains(actualRequest.URL.RawQuery, expectedQuery))
	}

	//assert.Equal(t, "api.opsgenie.com", actualRequest.URL.Host)
	assert.Equal(t, "/v2/integrations/maridv2/credentials", actualRequest.URL.Path)
}

func TestReceiveTokenInvalidJson(t *testing.T) {

	processor := newQueueProcessorTest()
	processor.retryer.DoFunc = mockHttpGetInvalidJson

	_, err := processor.receiveToken()

	assert.NotNil(t, err)
}

func TestReceiveTokenGetError(t *testing.T) {

	processor := newQueueProcessorTest()
	processor.retryer.DoFunc = mockHttpGetError

	_, err := processor.receiveToken()

	assert.NotNil(t, err)
	assert.Equal(t, "Test http error has occurred while getting token.", err.Error())
}

func TestReceiveTokenRequestError(t *testing.T) {

	defer func() {
		httpNewRequest = http.NewRequest
	}()

	processor := newQueueProcessorTest()
	httpNewRequest = func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, errors.New("Test: Http new request error.")
	}

	_, err := processor.receiveToken()

	assert.NotNil(t, err)
	assert.Equal(t, "Test: Http new request error.", err.Error())
}

func TestAddTwoDifferentPollersTest(t *testing.T) {

	processor := newQueueProcessorTest()

	poller := processor.addPoller(NewMockQueueProvider()).(*MaridPoller)
	processor.addPoller(&MaridQueueProvider{})

	assert.Equal(t, mockMaridMetadata1.getQueueUrl(), poller.QueueProvider().MaridMetadata().getQueueUrl())
	assert.Equal(t, processor.conf.PollerConf.PollingWaitIntervalInMillis, poller.pollerConf.PollingWaitIntervalInMillis)
	assert.Equal(t, processor.conf.PollerConf.MaxNumberOfMessages, poller.pollerConf.MaxNumberOfMessages)
	assert.Equal(t, processor.conf.PollerConf.VisibilityTimeoutInSeconds, poller.pollerConf.VisibilityTimeoutInSeconds)

	_, contains := processor.pollers[mockMaridMetadata1.getQueueUrl()]
	assert.True(t, contains)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRemovePollerTest(t *testing.T) {

	processor := newQueueProcessorTest()

	processor.pollers = mockPollers

	poller := processor.removePoller(mockQueueUrl1)
	processor.removePoller(mockQueueUrl2)

	assert.Equal(t, mockMaridMetadata1.getQueueUrl(), poller.QueueProvider().MaridMetadata().getQueueUrl())

	assert.Equal(t, 0, len(processor.pollers))
}

func TestRefreshPollersRepeat(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPoller = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshPollersAddAndRemove(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPoller = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockEmptyToken)

	assert.Equal(t, 0, len(processor.pollers))
}

func TestRefreshPollersAdd(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPoller = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockEmptyToken)
	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshPollersWithNotHavingPoller(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPoller = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshOldPollersAlreadyHavingPollers(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPoller = NewMockPollerForQueueProcessor
	processor.pollers = mockPollers

	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshPollersWithEmptyAssumeRoleResult(t *testing.T) {

	defer func() {
		newPoller = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPoller = NewMockPollerForQueueProcessor
	processor.pollers = mockPollers

	processor.refreshPollers(&mockTokenWithEmptyAssumeRoleResult)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshPollerWithEmptyToken(t *testing.T) {

	processor := newQueueProcessorTest()

	processor.refreshPollers(&mockEmptyToken)

	assert.Equal(t, 0, len(processor.pollers))
}

// Mock QueueProcessor

type MockQueueProcessor struct {

	StartProcessingFunc func() error
	StopProcessingFunc func() error
	IsRunningFunc func() bool
	WaitFunc func()
}

func NewMockQueueProcessor() *MockQueueProcessor {
	return &MockQueueProcessor{}
}

func (m *MockQueueProcessor) StartProcessing() error {
	if m.StartProcessingFunc != nil {
		return m.StartProcessingFunc()
	}
	return nil
}

func (m *MockQueueProcessor) StopProcessing() error {
	if m.StopProcessingFunc != nil {
		return m.StopProcessingFunc()
	}
	return nil
}

func (m *MockQueueProcessor)  IsRunning() bool {
	if m.IsRunningFunc != nil {
		return m.IsRunningFunc()
	}
	return false
}

func (m *MockQueueProcessor)  Wait() {
	if m.WaitFunc != nil {
		m.WaitFunc()
	}
}



