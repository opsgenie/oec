package queue

import (
	"bytes"
	"encoding/json"
	"github.com/opsgenie/ois/conf"
	"github.com/opsgenie/ois/git"
	"github.com/opsgenie/ois/retryer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var UserAgentHeader string

const (
	maxNumberOfWorker        = 12
	minNumberOfWorker        = 4
	queueSize                = 0
	keepAliveTimeInMillis    = 6000
	monitoringPeriodInMillis = 15000

	pollingWaitIntervalInMillis = 100
	visibilityTimeoutInSec      = 30
	maxNumberOfMessages         = 10

	successRefreshPeriod = time.Minute
	errorRefreshPeriod   = time.Minute

	repositoryRefreshPeriod = time.Minute
)

const tokenPath = "/v2/integrations/ois/credentials"

var newPollerFunc = NewPoller
var newRequestFunc = retryer.NewRequest

type QueueProcessor interface {
	StartProcessing() error
	StopProcessing() error
	IsRunning() bool
}

type OISQueueProcessor struct {
	workerPool WorkerPool
	pollers    map[string]Poller

	retryer *retryer.Retryer

	conf         *conf.Configuration
	repositories git.Repositories

	successRefreshPeriod time.Duration
	errorRefreshPeriod   time.Duration

	isRunning          bool
	isRunningWaitGroup *sync.WaitGroup
	startStopMutex     *sync.Mutex
	quit               chan struct{}
}

func NewQueueProcessor(conf *conf.Configuration) QueueProcessor {

	if conf.PollerConf.MaxNumberOfMessages <= 0 {
		logrus.Infof("Max number of messages should be greater than 0, default value[%d] is set.", maxNumberOfMessages)
		conf.PollerConf.MaxNumberOfMessages = maxNumberOfMessages
	}

	if conf.PollerConf.PollingWaitIntervalInMillis <= 0 {
		logrus.Infof("Polling wait interval should be greater than 0, default value[%d ms.] is set.", pollingWaitIntervalInMillis)
		conf.PollerConf.PollingWaitIntervalInMillis = pollingWaitIntervalInMillis
	}

	if conf.PollerConf.VisibilityTimeoutInSeconds < 15 {
		logrus.Infof("Visibility timeout cannot be lesser than 15 seconds or greater than 12 hours, default value[%d s.] is set.", visibilityTimeoutInSec)
		conf.PollerConf.VisibilityTimeoutInSeconds = visibilityTimeoutInSec
	}

	return &OISQueueProcessor{
		successRefreshPeriod: successRefreshPeriod,
		errorRefreshPeriod:   errorRefreshPeriod,
		workerPool:           NewWorkerPool(&conf.PoolConf),
		conf:                 conf,
		repositories:         git.NewRepositories(),
		pollers:              make(map[string]Poller),
		quit:                 make(chan struct{}),
		isRunning:            false,
		isRunningWaitGroup:   &sync.WaitGroup{},
		startStopMutex:       &sync.Mutex{},
		retryer:              &retryer.Retryer{},
	}
}

func (qp *OISQueueProcessor) IsRunning() bool {
	defer qp.startStopMutex.Unlock()
	qp.startStopMutex.Lock()

	return qp.isRunning
}

func (qp *OISQueueProcessor) StartProcessing() error {
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

	err = qp.repositories.DownloadAll(qp.conf.ActionMappings.GitActions())
	if err != nil {
		logrus.Errorf("Queue processor could not clone a git repository and will terminate.")
		return err
	}

	if qp.repositories.NotEmpty() {
		qp.isRunningWaitGroup.Add(1) // one for pulling repositories
		go qp.startPullingRepositories(repositoryRefreshPeriod)

		conf.AddRepositoryPathToGitActionFilepaths(qp.conf.ActionMappings, qp.repositories)
	}
	qp.workerPool.Start()
	qp.refreshPollers(token)
	qp.isRunningWaitGroup.Add(1) // one for receiving token
	go qp.run()

	qp.isRunning = true
	return nil
}

func (qp *OISQueueProcessor) StopProcessing() error {
	defer qp.startStopMutex.Unlock()
	qp.startStopMutex.Lock()

	if !qp.isRunning {
		return errors.New("Queue processor is not running.")
	}

	logrus.Infof("Queue processor is stopping.")

	close(qp.quit)
	qp.isRunningWaitGroup.Wait()

	qp.workerPool.Stop()
	qp.repositories.RemoveAll()

	qp.isRunning = false
	logrus.Infof("Queue processor has stopped.")
	return nil
}

func (qp *OISQueueProcessor) receiveToken() (*OISToken, error) {

	tokenUrl := qp.conf.BaseUrl + tokenPath

	request, err := newRequestFunc("GET", tokenUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "GenieKey "+qp.conf.ApiKey)
	request.Header.Add("X-OIS-Client-Info", UserAgentHeader)

	query := request.URL.Query()
	for _, poller := range qp.pollers {
		oisMetadata := poller.QueueProvider().OISMetadata()
		query.Add(
			oisMetadata.Region(),
			strconv.FormatInt(oisMetadata.ExpireTimeMillis(), 10),
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

	responseToken := bytes.NewBufferString(response.Header.Get("Token"))

	token := &OISToken{}
	err = json.NewDecoder(responseToken).Decode(&token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (qp *OISQueueProcessor) addPoller(queueProvider QueueProvider, integrationId string) Poller {
	poller := newPollerFunc(
		qp.workerPool,
		queueProvider,
		qp.conf,
		integrationId,
		qp.repositories,
	)
	qp.pollers[queueProvider.OISMetadata().QueueUrl()] = poller
	return poller
}

func (qp *OISQueueProcessor) removePoller(queueUrl string) Poller {
	poller := qp.pollers[queueUrl]
	delete(qp.pollers, queueUrl)
	return poller
}

func (qp *OISQueueProcessor) refreshPollers(token *OISToken) {
	pollerKeys := make(map[string]struct{}, len(qp.pollers))
	for key := range qp.pollers {
		pollerKeys[key] = struct{}{}
	}

	for _, oisMetadata := range token.OISMetadataList {
		queueUrl := oisMetadata.QueueUrl()

		// refresh existing pollers if there comes new assumeRoleResult
		if poller, contains := qp.pollers[queueUrl]; contains {
			isTokenRefreshed := oisMetadata.AssumeRoleResult != AssumeRoleResult{}
			if isTokenRefreshed {
				err := poller.RefreshClient(oisMetadata.AssumeRoleResult)
				if err != nil {
					logrus.Errorf("Client of queue provider[%s] could not be refreshed.", queueUrl)
				}
				logrus.Infof("Client of queue provider[%s] has refreshed.", queueUrl)
			}
			delete(pollerKeys, queueUrl)

			// add new pollers
		} else {
			queueProvider, err := NewQueueProvider(oisMetadata)
			if err != nil {
				logrus.Errorf("Poller[%s] could not be added: %s.", queueUrl, err)
				continue
			}
			qp.addPoller(queueProvider, token.IntegrationId).StartPolling()
			logrus.Debugf("Poller[%s] is added.", queueUrl)
		}
	}

	// remove unnecessary pollers
	for queueUrl := range pollerKeys {
		qp.removePoller(queueUrl).StopPolling()
		logrus.Debugf("Poller[%s] is removed.", queueUrl)
	}

	if len(token.OISMetadataList) != 0 { // pick first oisMetadata to refresh waitPeriods, can be change for further usage
		qp.successRefreshPeriod = time.Second * time.Duration(token.OISMetadataList[0].QueueConfiguration.SuccessRefreshPeriodInSeconds)
		qp.errorRefreshPeriod = time.Second * time.Duration(token.OISMetadataList[0].QueueConfiguration.ErrorRefreshPeriodInSeconds)
	}
}

func (qp *OISQueueProcessor) run() {

	logrus.Infof("Queue processor has started to run. Refresh client period: %s.", qp.successRefreshPeriod.String())

	ticker := time.NewTicker(qp.successRefreshPeriod)

	for {
		select {
		case <-qp.quit:
			ticker.Stop()
			for _, poller := range qp.pollers {
				poller.StopPolling()
			}
			qp.isRunningWaitGroup.Done()
			return
		case <-ticker.C:
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

func (qp *OISQueueProcessor) startPullingRepositories(pullPeriod time.Duration) {

	logrus.Infof("Repositories will be updated in every %s.", pullPeriod.String())

	ticker := time.NewTicker(pullPeriod)

	for {
		select {
		case <-qp.quit:
			ticker.Stop()
			logrus.Info("All git repositories are removed.")
			qp.isRunningWaitGroup.Done()
			return
		case <-ticker.C:
			ticker.Stop()
			qp.repositories.PullAll()
			ticker = time.NewTicker(pullPeriod)
		}
	}
}
