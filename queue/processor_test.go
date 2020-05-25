package queue

import (
	"bytes"
	"encoding/json"
	"github.com/opsgenie/oec/conf"
	"github.com/opsgenie/oec/git"
	"github.com/opsgenie/oec/retryer"
	"github.com/opsgenie/oec/worker_pool"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var mockConf = &conf.Configuration{
	ApiKey:     "ApiKey",
	PollerConf: *mockPollerConf,
	PoolConf:   *mockPoolConf,
}

var mockPoolConf = &conf.PoolConf{
	MaxNumberOfWorker: 16,
	MinNumberOfWorker: 2,
}

func newQueueProcessorTest() *processor {

	return &processor{
		successRefreshPeriod: successRefreshPeriod,
		errorRefreshPeriod:   errorRefreshPeriod,
		workerPool:           NewMockWorkerPool(),
		configuration:        mockConf,
		repositories:         git.NewRepositories(),
		pollers:              make(map[string]Poller),
		quit:                 make(chan struct{}),
		isRunning:            false,
		isRunningWg:          &sync.WaitGroup{},
		startStopMu:          &sync.Mutex{},
		retryer:              &retryer.Retryer{},
	}
}

var mockPollers = map[string]Poller{
	mockQueueUrl1: NewMockPoller(),
	mockQueueUrl2: NewMockPoller(),
}

func mockHttpGetError(retryer *retryer.Retryer, request *retryer.Request) (*http.Response, error) {
	return nil, errors.New("Test http error has occurred while getting token.")
}

func mockHttpGet(retryer *retryer.Retryer, request *retryer.Request) (*http.Response, error) {

	token, _ := json.Marshal(mockToken)

	header := http.Header{}
	header.Add("Token", string(token))

	response := &http.Response{
		StatusCode: 200,
		Header:     header,
		Body:       ioutil.NopCloser(nil),
	}

	return response, nil
}

func mockHttpGetInvalidJson(retryer *retryer.Retryer, request *retryer.Request) (*http.Response, error) {

	response := &http.Response{}
	response.Body = ioutil.NopCloser(bytes.NewBufferString(`{"Invalid json": }`))

	return response, nil
}

func TestValidateNewQueueProcessor(t *testing.T) {
	configuration := &conf.Configuration{}
	processor := NewProcessor(configuration).(*processor)

	assert.Equal(t, int64(maxNumberOfMessages), processor.configuration.PollerConf.MaxNumberOfMessages)
	assert.Equal(t, int64(visibilityTimeoutInSec), processor.configuration.PollerConf.VisibilityTimeoutInSeconds)
	assert.Equal(t, time.Duration(pollingWaitIntervalInMillis), processor.configuration.PollerConf.PollingWaitIntervalInMillis)
}

func TestStartAndStopQueueProcessor(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	processor.retryer.DoFunc = mockHttpGet
	newPollerFunc = NewMockPollerForQueueProcessor

	err := processor.Start()
	assert.Nil(t, err)

	assert.Equal(t, 2, len(processor.pollers))

	err = processor.Stop()
	assert.Nil(t, err)
}

func TestStartQueueProcessorAndRefresh(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	processor.retryer.DoFunc = mockHttpGet
	processor.successRefreshPeriod = time.Nanosecond
	newPollerFunc = NewMockPollerForQueueProcessor

	err := processor.Start()
	assert.Nil(t, err)

	time.Sleep(time.Nanosecond * 100)

	assert.Equal(t, 2, len(processor.pollers))
	assert.Equal(t, successRefreshPeriod, processor.successRefreshPeriod)

	err = processor.Stop()
	assert.Nil(t, err)
}

func TestStartQueueProcessorInitialError(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	processor.retryer.DoFunc = mockHttpGetError
	newPollerFunc = NewMockPollerForQueueProcessor

	err := processor.Start()

	assert.NotNil(t, err)
	assert.Equal(t, "Test http error has occurred while getting token.", err.Error())
}

func TestStopQueueProcessorWhileNotRunning(t *testing.T) {

	processor := newQueueProcessorTest()

	err := processor.Stop()

	assert.NotNil(t, err)
	assert.Equal(t, "Queue processor is not running.", err.Error())
}

func TestReceiveToken(t *testing.T) {

	processor := newQueueProcessorTest()

	processor.pollers = mockPollers

	var actualRequest *http.Request

	processor.retryer.DoFunc = func(retryer *retryer.Retryer, request *retryer.Request) (*http.Response, error) {
		actualRequest = request.Request
		return mockHttpGet(retryer, request)
	}

	token, err := processor.receiveToken()

	assert.Nil(t, err)
	assert.Equal(t, 2, len(token.QueuePropertiesList))
	assert.Equal(t, "accessKeyId1", token.QueuePropertiesList[0].AssumeRoleResult.Credentials.AccessKeyId)
	assert.Equal(t, "accessKeyId2", token.QueuePropertiesList[1].AssumeRoleResult.Credentials.AccessKeyId)

	for _, poller := range processor.pollers {
		queueProperties := poller.QueueProvider().Properties()
		expectedQuery := queueProperties.Region() + "=" + strconv.FormatInt(queueProperties.ExpireTimeMillis(), 10)

		assert.True(t, strings.Contains(actualRequest.URL.RawQuery, expectedQuery))
	}

	//assert.Equal(t, "api.opsgenie.com", actualRequest.URL.Host)
	assert.Equal(t, "/v2/integrations/oec/credentials", actualRequest.URL.Path)
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

	processor := newQueueProcessorTest()
	processor.configuration.BaseUrl = "invalid"

	_, err := processor.receiveToken()

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid"+tokenPath+"")
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestAddTwoDifferentPollersTest(t *testing.T) {

	processor := newQueueProcessorTest()

	p1, _ := processor.addPoller(mockQueueProperties1, mockOwnerId)
	poller1 := p1.(*poller)

	mockQueueProvider2 := NewMockQueueProvider().(*MockSQSProvider)
	mockQueueProvider2.QueuePropertiesFunc = func() Properties {
		return mockQueueProperties2
	}

	processor.addPoller(mockQueueProperties2, mockOwnerId)

	assert.Equal(t, mockQueueProperties1, poller1.QueueProvider().Properties())
	assert.Equal(t, processor.configuration.PollerConf, poller1.conf.PollerConf)

	_, contains := processor.pollers[mockQueueProperties1.Url()]
	assert.True(t, contains)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRemovePollerTest(t *testing.T) {

	processor := newQueueProcessorTest()

	processor.pollers = mockPollers

	poller := processor.removePoller(mockQueueUrl1)
	processor.removePoller(mockQueueUrl2)

	assert.Equal(t, mockQueueProperties1.Url(), poller.QueueProvider().Properties().Url())

	assert.Equal(t, 0, len(processor.pollers))
}

func TestRefreshPollersRepeat(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPollerFunc = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshPollersAddAndRemove(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPollerFunc = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockEmptyToken)

	assert.Equal(t, 0, len(processor.pollers))
}

func TestRefreshPollersAdd(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPollerFunc = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockEmptyToken)
	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshPollersWithNotHavingPoller(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPollerFunc = NewMockPollerForQueueProcessor

	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)
	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshOldPollersAlreadyHavingPollers(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPollerFunc = NewMockPollerForQueueProcessor
	processor.pollers = mockPollers

	processor.refreshPollers(&mockToken)

	assert.Equal(t, 2, len(processor.pollers))
}

func TestRefreshPollersWithEmptyAssumeRoleResult(t *testing.T) {

	defer func() {
		newPollerFunc = NewPoller
	}()

	processor := newQueueProcessorTest()

	newPollerFunc = NewMockPollerForQueueProcessor
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
	StopProcessingFunc  func() error
	IsRunningFunc       func() bool
	WaitFunc            func()
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

func (m *MockQueueProcessor) IsRunning() bool {
	if m.IsRunningFunc != nil {
		return m.IsRunningFunc()
	}
	return false
}

func (m *MockQueueProcessor) Wait() {
	if m.WaitFunc != nil {
		m.WaitFunc()
	}
}

// Mock Worker Pool
type MockWorkerPool struct {
	NumberOfAvailableWorkerFunc func() int32
	StartFunc                   func() error
	StopFunc                    func() error
	SubmitFunc                  func(worker_pool.Job) (bool, error)
}

func NewMockWorkerPool() *MockWorkerPool {
	return &MockWorkerPool{}
}

func (m *MockWorkerPool) NumberOfAvailableWorker() int32 {
	if m.NumberOfAvailableWorkerFunc != nil {
		return m.NumberOfAvailableWorkerFunc()
	}
	return 0
}

func (m *MockWorkerPool) Start() error {
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

func (m *MockWorkerPool) Stop() error {
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

func (m *MockWorkerPool) Submit(job worker_pool.Job) (bool, error) {
	if m.SubmitFunc != nil {
		return m.SubmitFunc(job)
	}
	return false, nil
}
