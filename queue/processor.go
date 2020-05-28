package queue

import (
	"bytes"
	"encoding/json"
	"github.com/opsgenie/oec/conf"
	"github.com/opsgenie/oec/git"
	"github.com/opsgenie/oec/retryer"
	"github.com/opsgenie/oec/worker_pool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var UserAgentHeader string

const (
	pollingWaitIntervalInMillis = 100
	visibilityTimeoutInSec      = 30
	maxNumberOfMessages         = 10

	successRefreshPeriod = time.Minute
	errorRefreshPeriod   = time.Minute

	repositoryRefreshPeriod = time.Minute
)

const tokenPath = "/v2/integrations/oec/credentials"

var newPollerFunc = NewPoller

type Processor interface {
	Start() error
	Stop() error
}

type processor struct {
	workerPool worker_pool.WorkerPool
	pollers    map[string]Poller

	retryer *retryer.Retryer

	configuration *conf.Configuration
	repositories  git.Repositories
	actionLoggers map[string]io.Writer

	successRefreshPeriod time.Duration
	errorRefreshPeriod   time.Duration

	isRunning   bool
	isRunningWg *sync.WaitGroup
	startStopMu *sync.Mutex
	quit        chan struct{}
}

func NewProcessor(conf *conf.Configuration) Processor {

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

	return &processor{
		successRefreshPeriod: successRefreshPeriod,
		errorRefreshPeriod:   errorRefreshPeriod,
		workerPool:           worker_pool.New(&conf.PoolConf),
		configuration:        conf,
		repositories:         git.NewRepositories(),
		actionLoggers:        newActionLoggers(conf.ActionMappings),
		pollers:              make(map[string]Poller),
		quit:                 make(chan struct{}),
		isRunning:            false,
		isRunningWg:          &sync.WaitGroup{},
		startStopMu:          &sync.Mutex{},
		retryer:              &retryer.Retryer{},
	}
}

func (qp *processor) Start() error {
	defer qp.startStopMu.Unlock()
	qp.startStopMu.Lock()

	if qp.isRunning {
		return errors.New("Queue processor is already running.")
	}

	logrus.Infof("Queue processor is starting.")
	token, err := qp.receiveToken()
	if err != nil {
		logrus.Errorf("Queue processor could not get initial token and will terminate.")
		return err
	}

	err = qp.repositories.DownloadAll(qp.configuration.ActionMappings.GitActions())
	if err != nil {
		logrus.Errorf("Queue processor could not clone a git repository and will terminate.")
		return err
	}

	if qp.repositories.NotEmpty() {
		qp.isRunningWg.Add(1) // one for pulling repositories
		go qp.startPullingRepositories(repositoryRefreshPeriod)

		conf.AddRepositoryPathToGitActionFilepaths(qp.configuration.ActionMappings, qp.repositories)
	}
	qp.workerPool.Start()
	qp.refreshPollers(token)
	qp.isRunningWg.Add(1) // one for receiving token
	go qp.run()

	qp.isRunning = true
	return nil
}

func (qp *processor) Stop() error {
	defer qp.startStopMu.Unlock()
	qp.startStopMu.Lock()

	if !qp.isRunning {
		return errors.New("Queue processor is not running.")
	}

	logrus.Infof("Queue processor is stopping.")

	close(qp.quit)
	qp.isRunningWg.Wait()

	qp.workerPool.Stop()
	qp.repositories.RemoveAll()

	qp.isRunning = false
	logrus.Infof("Queue processor has stopped.")
	return nil
}

func (qp *processor) receiveToken() (*token, error) {

	tokenUrl := qp.configuration.BaseUrl + tokenPath

	request, err := retryer.NewRequest(http.MethodGet, tokenUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "GenieKey "+qp.configuration.ApiKey)
	request.Header.Add("X-OEC-Client-Info", UserAgentHeader)

	query := request.URL.Query()
	for _, poller := range qp.pollers {
		queueProperties := poller.QueueProvider().Properties()
		query.Add(
			queueProperties.Region(),
			strconv.FormatInt(queueProperties.ExpireTimeMillis(), 10),
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

	token := &token{}
	err = json.NewDecoder(responseToken).Decode(&token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (qp *processor) addPoller(queueProperties Properties, ownerId string) (Poller, error) {

	queueProvider, err := NewSqsProvider(queueProperties)
	if err != nil {
		return nil, err
	}

	messageHandler := &messageHandler{
		repositories:  qp.repositories,
		actionSpecs:   qp.configuration.ActionSpecifications,
		actionLoggers: qp.actionLoggers,
	}

	poller := newPollerFunc(
		qp.workerPool,
		queueProvider,
		messageHandler,
		qp.configuration,
		ownerId,
	)
	qp.pollers[queueProvider.Properties().Url()] = poller
	return poller, nil
}

func (qp *processor) removePoller(queueUrl string) Poller {
	poller := qp.pollers[queueUrl]
	delete(qp.pollers, queueUrl)
	return poller
}

func (qp *processor) refreshPollers(token *token) {
	pollerKeys := make(map[string]struct{}, len(qp.pollers))
	for key := range qp.pollers {
		pollerKeys[key] = struct{}{}
	}

	for _, queueProperties := range token.QueuePropertiesList {
		queueUrl := queueProperties.Url()

		// refresh existing pollers if there comes new AssumeRoleResult
		if poller, contains := qp.pollers[queueUrl]; contains {
			isTokenRefreshed := queueProperties.AssumeRoleResult != AssumeRoleResult{}
			if isTokenRefreshed {
				err := poller.RefreshClient(queueProperties.AssumeRoleResult)
				if err != nil {
					logrus.Errorf("Client of queue provider[%s] could not be refreshed.", queueUrl)
				}
				logrus.Infof("Client of queue provider[%s] has refreshed.", queueUrl)
			}
			delete(pollerKeys, queueUrl)

			// add new pollers
		} else {
			poller, err := qp.addPoller(queueProperties, token.OwnerId)
			if err != nil {
				logrus.Errorf("Poller[%s] could not be added: %s.", queueUrl, err)
				continue
			}
			poller.Start()
			logrus.Debugf("Poller[%s] is added.", queueUrl)
		}
	}

	// remove unnecessary pollers
	for queueUrl := range pollerKeys {
		qp.removePoller(queueUrl).Stop()
		logrus.Debugf("Poller[%s] is removed.", queueUrl)
	}

	if len(token.QueuePropertiesList) != 0 { // pick first Properties to refresh waitPeriods, can be change for further usage
		qp.successRefreshPeriod = time.Second * time.Duration(token.QueuePropertiesList[0].Configuration.SuccessRefreshPeriodInSeconds)
		qp.errorRefreshPeriod = time.Second * time.Duration(token.QueuePropertiesList[0].Configuration.ErrorRefreshPeriodInSeconds)
	}
}

func (qp *processor) run() {

	logrus.Infof("Queue processor has started to run. Refresh client period: %s.", qp.successRefreshPeriod.String())

	ticker := time.NewTicker(qp.successRefreshPeriod)

	for {
		select {
		case <-qp.quit:
			ticker.Stop()
			for _, poller := range qp.pollers {
				poller.Stop()
			}
			qp.isRunningWg.Done()
			return
		case <-ticker.C:
			ticker.Stop()
			token, err := qp.receiveToken()
			if err != nil {
				logrus.Warnf("Refresh cycle of queue processor has failed: %s", err)
				logrus.Debugf("Will refresh token after %s", qp.errorRefreshPeriod.String())
				ticker = time.NewTicker(qp.errorRefreshPeriod)
				break
			}
			qp.refreshPollers(token)

			ticker = time.NewTicker(qp.successRefreshPeriod)
		}
	}
}

func (qp *processor) startPullingRepositories(pullPeriod time.Duration) {

	logrus.Infof("Repositories will be updated in every %s.", pullPeriod.String())

	ticker := time.NewTicker(pullPeriod)

	for {
		select {
		case <-qp.quit:
			ticker.Stop()
			logrus.Info("All git repositories will be removed.")
			qp.isRunningWg.Done()
			return
		case <-ticker.C:
			ticker.Stop()
			qp.repositories.PullAll()
			ticker = time.NewTicker(pullPeriod)
		}
	}
}

func newActionLoggers(mappings conf.ActionMappings) map[string]io.Writer {
	actionLoggers := make(map[string]io.Writer)
	for _, action := range mappings {
		if action.Stdout != "" {
			if _, ok := actionLoggers[action.Stdout]; !ok {
				actionLoggers[action.Stdout] = newLogger(action.Stdout)
			}
		}
		if action.Stderr != "" {
			if _, ok := actionLoggers[action.Stderr]; !ok {
				actionLoggers[action.Stderr] = newLogger(action.Stderr)
			}
		}
	}
	return actionLoggers
}

func newLogger(filename string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:  filename,
		MaxSize:   3, // MB
		MaxAge:    1, // Days
		LocalTime: true,
	}
}
