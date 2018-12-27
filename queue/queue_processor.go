package queue

import (
	"encoding/json"
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/retryer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxNumberOfWorker = 12
	minNumberOfWorker = 4
	queueSize = 0
	keepAliveTimeInMillis = 6000
	monitoringPeriodInMillis = 15000

	pollingWaitIntervalInMillis = 100
	visibilityTimeoutInSec = 30
	maxNumberOfMessages = 10

	successRefreshPeriod = time.Minute
	errorRefreshPeriod = time.Minute
)

const tokenPath = "/v2/integrations/maridv2/credentials"

var newPollerFunc = NewPoller
var httpNewRequestFunc = http.NewRequest

type QueueProcessor interface {

	StartProcessing() error
	StopProcessing() error
	IsRunning() bool
}

type MaridQueueProcessor struct {

	successRefreshPeriod 	time.Duration
	errorRefreshPeriod 		time.Duration

	conf *conf.Configuration

	workerPool  WorkerPool
	pollers 	map[string]Poller

	isRunning          bool
	isRunningWaitGroup *sync.WaitGroup
	startStopMutex     *sync.Mutex

	retryer *retryer.Retryer
	quit    chan struct{}
}

func NewQueueProcessor(conf *conf.Configuration) QueueProcessor {

	if conf.PollerConf.MaxNumberOfMessages <= 0 {
		logrus.Infof("Max number of messages should be greater than 0, default value[%d] is set.", maxNumberOfMessages)
		conf.PollerConf.MaxNumberOfMessages = maxNumberOfMessages
	}

	if conf.PollerConf.PollingWaitIntervalInMillis <= 0 {
		logrus.Infof("Max number of messages should be greater than 0, default value[%d] is set.", maxNumberOfMessages)
		conf.PollerConf.PollingWaitIntervalInMillis = pollingWaitIntervalInMillis
	}

	if conf.PollerConf.VisibilityTimeoutInSeconds < 15 {
		logrus.Infof("Visibility timeout cannot be lesser than 15 seconds or greater than 12 hours, default value[%d ms.] is set.", pollingWaitIntervalInMillis)
		conf.PollerConf.VisibilityTimeoutInSeconds = visibilityTimeoutInSec
	}

	return &MaridQueueProcessor{
		successRefreshPeriod:       successRefreshPeriod,
		errorRefreshPeriod:         errorRefreshPeriod,
		workerPool:					NewWorkerPool(&conf.PoolConf),
		conf:						conf,
		pollers:                    make(map[string]Poller),
		quit:                       make(chan struct{}),
		isRunning:					false,
		isRunningWaitGroup:         &sync.WaitGroup{},
		startStopMutex:             &sync.Mutex{},
		retryer:                    &retryer.Retryer{},
	}
}

func (qp *MaridQueueProcessor) IsRunning() bool {
	defer qp.startStopMutex.Unlock()
	qp.startStopMutex.Lock()

	return qp.isRunning
}

func (qp *MaridQueueProcessor) StartProcessing() error {
	defer qp.startStopMutex.Unlock()
	qp.startStopMutex.Lock()

	if qp.isRunning {
		return errors.New("Queue processor is already running.")
	}

	logrus.Infof("Queue processor is starting.")
	token, err := qp.receiveToken()
	if err != nil {
		logrus.Errorf("Queue processor could not get initial token and will terminate.")
		return err
	}

	props := strings.SplitAfter(token.Data.MaridMetaDataList[0].QueueUrl(), "marid-")[1]
	ids := strings.Split(props, "-")
	logrus.Infof("Initial token is received. CustomerId: %s, IntegrationId: %s", ids[0], ids[1])

	qp.workerPool.Start()
	qp.refreshPollers(token)
	go qp.run()

	qp.isRunning = true
	qp.isRunningWaitGroup.Add(1)
	return nil
}

func (qp *MaridQueueProcessor) StopProcessing() error {
	defer qp.startStopMutex.Unlock()
	qp.startStopMutex.Lock()

	if !qp.isRunning {
		return errors.New("Queue processor is not running.")
	}

	logrus.Infof("Queue processor is stopping.")

	close(qp.quit)
	qp.workerPool.Stop()

	qp.isRunning = false
	qp.isRunningWaitGroup.Wait()
	logrus.Infof("Queue processor has stopped.")
	return nil
}

func (qp *MaridQueueProcessor) receiveToken() (*MaridToken, error) {

	tokenUrl := qp.conf.BaseUrl + tokenPath

	request, err := httpNewRequestFunc("GET", tokenUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "GenieKey " + qp.conf.ApiKey)
	// todo add user agent (version etc.)

	query := request.URL.Query()
	for _, poller := range qp.pollers {
		maridMetadata := poller.QueueProvider().MaridMetadata()
		query.Add(
			maridMetadata.Region(),
			strconv.FormatInt(maridMetadata.ExpireTimeMillis(), 10),
		)
	}
	request.URL.RawQuery = query.Encode()

	response, err := qp.retryer.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, errors.Errorf("Token could not be received from Opsgenie, status: %s, message: %s", response.Status, body)
	}

	token := &MaridToken{}
	err = json.NewDecoder(response.Body).Decode(&token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (qp *MaridQueueProcessor) addPoller(queueProvider QueueProvider) Poller {
	poller := newPollerFunc(
		qp.workerPool,
		queueProvider,
		&qp.conf.PollerConf,
		&qp.conf.ActionMappings,
		&qp.conf.ApiKey,
		&qp.conf.BaseUrl,
	)
	qp.pollers[queueProvider.MaridMetadata().QueueUrl()] = poller
	return poller
}

func (qp *MaridQueueProcessor) removePoller(queueUrl string) Poller {
	poller := qp.pollers[queueUrl]
	delete(qp.pollers, queueUrl)
	return poller
}

func (qp *MaridQueueProcessor) refreshPollers(token *MaridToken) {
	pollerKeys := make(map[string]struct{}, len(qp.pollers))
	for key := range qp.pollers {
		pollerKeys[key] = struct{}{}
	}

	for _, maridMetadata := range token.Data.MaridMetaDataList {
		queueUrl := maridMetadata.QueueUrl()
		if poller, contains := qp.pollers[queueUrl]; contains {
			isTokenRefreshed := maridMetadata.AssumeRoleResult != AssumeRoleResult{}
			if isTokenRefreshed {
				poller.RefreshClient(maridMetadata.AssumeRoleResult)
			}
			delete(pollerKeys, queueUrl)
		} else {
			queueProvider, err := NewQueueProvider(maridMetadata, token.Data.IntegrationId)
			if err != nil {
				logrus.Errorf("Poller[%s] could not be added: %s.", queueUrl, err)
				continue
			}
			qp.addPoller(queueProvider).StartPolling()
			logrus.Debugf("Poller[%s] is added.", queueUrl)
		}
	}
	for queueUrl := range pollerKeys {
		qp.removePoller(queueUrl).StopPolling()
		logrus.Debugf("Poller[%s] is removed.", queueUrl)
	}

	if len(token.Data.MaridMetaDataList) != 0 {
		qp.successRefreshPeriod = time.Second * time.Duration(token.Data.MaridMetaDataList[0].QueueConfiguration.SuccessRefreshPeriodInSeconds)
		qp.errorRefreshPeriod = time.Second * time.Duration(token.Data.MaridMetaDataList[0].QueueConfiguration.ErrorRefreshPeriodInSeconds)
	}
}

func (qp *MaridQueueProcessor) run() {

	logrus.Infof("Queue processor has started to run. Refresh client period: %s.", qp.successRefreshPeriod.String())

	ticker := time.NewTicker(qp.successRefreshPeriod)

	for {
		select {
		case <- qp.quit:
			ticker.Stop()
			for _, poller := range qp.pollers {
				poller.StopPolling()
			}
			qp.isRunningWaitGroup.Done()
			return
		case <- ticker.C:
			ticker.Stop()
			token, err := qp.receiveToken()
			if err != nil {
				logrus.Warnf("Refresh cycle of queue processor has failed: %s", err)
				logrus.Infof("Will refresh token after %s", qp.errorRefreshPeriod.String())
				ticker = time.NewTicker(qp.errorRefreshPeriod)
				break
			}
			qp.refreshPollers(token)

			ticker = time.NewTicker(qp.successRefreshPeriod)
		}
	}
}