package queue

import (
	"github.com/pkg/errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
	"github.com/opsgenie/marid2/conf"
	"encoding/json"
	"strconv"
)

const maxNumberOfWorker = 50
const minNumberOfWorker = 15
const queueSize = 0
const keepAliveTime = time.Second
const monitoringPeriod = time.Second * 1

const pollingWaitInterval = time.Second * 10
const visibilityTimeoutInSeconds = 30
const maxNumberOfMessages = 10

const successRefreshPeriod = time.Minute
const errorRefreshPeriod = time.Minute

type QueueProcessor interface {
	Start() error
	Stop() error

	Wait()
	IsRunning() bool

	setMaxNumberOfWorker(max uint32) QueueProcessor
	setMinNumberOfWorker(max uint32) QueueProcessor
	setQueueSize(queueSize uint32) QueueProcessor
	setKeepAliveTime(keepAliveTime time.Duration) QueueProcessor
	setMonitoringPeriod(monitoringPeriod time.Duration) QueueProcessor

	setPollingWaitInterval(interval time.Duration) QueueProcessor
	setMaxNumberOfMessages(max int64) QueueProcessor
	setMessageVisibilityTimeout(timeoutInSeconds int64) QueueProcessor
}

type MaridQueueProcessor struct {

	successRefreshPeriod 	time.Duration
	errorRefreshPeriod 		time.Duration

	pollingWaitInterval 		time.Duration
	visibilityTimeoutInSeconds 	int64
	maxNumberOfMessages 		int64

	workerPool  WorkerPool
	pollers 	map[string]Poller

	isRunning   atomic.Value
	startStopMu *sync.Mutex
	wg        	*sync.WaitGroup

	token   *MaridToken
	retryer *Retryer
	quit    chan struct{}

	StartMethod func(qp *MaridQueueProcessor) 	error
	StopMethod func(qp *MaridQueueProcessor) 	error

	runMethod           	func(qp *MaridQueueProcessor)
	receiveTokenMethod  	func(qp *MaridQueueProcessor) (*MaridToken, error)
	addPollerMethod 		func(qp *MaridQueueProcessor, queueProvider QueueProvider) Poller
	removePollerMethod 		func(qp *MaridQueueProcessor, queueUrl string) Poller
	refreshPollersMethod	func(qp *MaridQueueProcessor, token *MaridToken)
}

func NewQueueProcessor() QueueProcessor {
	qp := &MaridQueueProcessor{
		quit:                       make(chan struct{}),
		startStopMu:                &sync.Mutex{},
		wg:                         &sync.WaitGroup{},
		retryer:                    NewRetryer(),
		pollers:					make(map[string]Poller),
		StartMethod:                Start,
		StopMethod:                 Stop,
		receiveTokenMethod:         receiveToken,
		runMethod:                  runQueueProcessor,
		addPollerMethod:            addPoller,
		removePollerMethod:         removePoller,
		refreshPollersMethod:       refreshPollers,
		successRefreshPeriod:		successRefreshPeriod,
		errorRefreshPeriod:			errorRefreshPeriod,
		pollingWaitInterval:        pollingWaitInterval,
		visibilityTimeoutInSeconds: visibilityTimeoutInSeconds,
		maxNumberOfMessages:        maxNumberOfMessages,

	}
	qp.isRunning.Store(false)
	qp.workerPool = NewWorkerPool(maxNumberOfWorker, minNumberOfWorker, queueSize, keepAliveTime, monitoringPeriod)

	return qp
}

func (qp *MaridQueueProcessor) Wait() {
	qp.wg.Wait()
}

func (qp *MaridQueueProcessor) Start() error {
	return qp.StartMethod(qp)
}

func (qp *MaridQueueProcessor) Stop() error {
	return qp.StopMethod(qp)
}

func (qp *MaridQueueProcessor) run() {
	go qp.runMethod(qp)
}

func (qp *MaridQueueProcessor) receiveToken() (*MaridToken, error) {
	return qp.receiveTokenMethod(qp)
}

func (qp *MaridQueueProcessor) addPoller(queueProvider QueueProvider) Poller {
	return qp.addPollerMethod(qp, queueProvider)
}

func (qp *MaridQueueProcessor) removePoller(queueUrl string) Poller {
	return qp.removePollerMethod(qp, queueUrl)
}

func (qp *MaridQueueProcessor) refreshPollers(token *MaridToken) {
	qp.refreshPollersMethod(qp, token)
}

func (qp *MaridQueueProcessor) IsRunning() bool {
	return qp.isRunning.Load().(bool)
}

func (qp *MaridQueueProcessor) setMaxNumberOfWorker(max uint32) QueueProcessor {
	qp.workerPool.SetMaxNumberOfWorker(max)
	return qp
}

func (qp *MaridQueueProcessor) setMinNumberOfWorker(min uint32) QueueProcessor {
	qp.workerPool.SetMinNumberOfWorker(min)
	return qp
}

func (qp *MaridQueueProcessor) setQueueSize(queueSize uint32) QueueProcessor {
	qp.workerPool.SetQueueSize(queueSize)
	return qp
}

func (qp *MaridQueueProcessor) setKeepAliveTime(keepAliveTime time.Duration) QueueProcessor {
	qp.workerPool.SetKeepAliveTime(keepAliveTime)
	return qp
}

func (qp *MaridQueueProcessor) setMonitoringPeriod(monitoringPeriod time.Duration) QueueProcessor {
	qp.workerPool.SetMonitoringPeriod(monitoringPeriod)
	return qp
}

func (qp *MaridQueueProcessor) setPollingWaitInterval(pollingWaitInterval time.Duration) QueueProcessor {
	qp.pollingWaitInterval = pollingWaitInterval
	return qp
}

func (qp *MaridQueueProcessor) setMaxNumberOfMessages(maxNumberOfMessages int64) QueueProcessor {
	qp.maxNumberOfMessages = maxNumberOfMessages
	return qp
}

func (qp *MaridQueueProcessor) setMessageVisibilityTimeout(visibilityTimeoutInSeconds int64) QueueProcessor {
	qp.visibilityTimeoutInSeconds = visibilityTimeoutInSeconds
	return qp
}

func Start(qp *MaridQueueProcessor) error {
	defer qp.startStopMu.Unlock()
	qp.startStopMu.Lock()

	if qp.isRunning.Load().(bool) {
		return errors.New("Queue processor is already running.")
	}

	log.Printf("Queue processor is starting.")
	token, err := qp.receiveToken()
	if err != nil {
		log.Printf("Queue processor could not get initial token and will terminate.")
		return err
	}

	qp.workerPool.Start()
	qp.refreshPollers(token)
	qp.run()

	qp.isRunning.Store(true)
	qp.wg.Add(1)

	return nil
}

func Stop(qp *MaridQueueProcessor) error {
	defer qp.startStopMu.Unlock()
	qp.startStopMu.Lock()

	if !qp.isRunning.Load().(bool) {
		return errors.New("Queue processor is not running.")
	}

	log.Println("Queue processor is stopping.")

	close(qp.quit)
	qp.workerPool.Stop()

	qp.isRunning.Store(false)

	qp.wg.Done()
	return nil
}

func receiveToken(qp *MaridQueueProcessor) (*MaridToken, error) {

	request, err := httpNewRequest("GET", tokenUrl, nil)
	if err != nil {
		return nil, err
	}
	apiKey, ok := conf.Configuration["apiKey"].(string)
	if !ok {
		return nil, errors.New("The configuration does not have an api key.")
	}
	request.Header.Add("Authorization", "GenieKey " + apiKey)

	query := request.URL.Query()
	for _, poller := range qp.pollers {
		maridMetadata := poller.GetQueueProvider().GetMaridMetadata()
		query.Add(
			maridMetadata.getRegion(),
			strconv.FormatInt(maridMetadata.getExpireTimeMillis(), 10),
		)
	}
	request.URL.RawQuery = query.Encode()

	response, err := qp.retryer.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	token := &MaridToken{}
	err = json.NewDecoder(response.Body).Decode(&token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func addPoller(qp *MaridQueueProcessor, queueProvider QueueProvider) Poller {
	poller := NewPoller(
		qp.workerPool,
		queueProvider,
		&qp.pollingWaitInterval,
		&qp.maxNumberOfMessages,
		&qp.visibilityTimeoutInSeconds,
	)
	qp.pollers[queueProvider.GetMaridMetadata().getQueueUrl()] = poller
	return poller
}

func removePoller(qp *MaridQueueProcessor, queueUrl string) Poller {
	poller := qp.pollers[queueUrl]
	delete(qp.pollers, queueUrl)
	return poller
}

func refreshPollers(qp *MaridQueueProcessor, token *MaridToken) {
	pollerKeys := make(map[string]struct{}, len(qp.pollers))
	for key := range qp.pollers {
		pollerKeys[key] = struct{}{}
	}

	for _, maridMetadata := range token.MaridMetaDataList {
		queueUrl := maridMetadata.getQueueUrl()
		if _, contains := pollerKeys[queueUrl]; contains {
			isTokenRefreshed := maridMetadata.AssumeRoleResult != AssumeRoleResult{}
			if isTokenRefreshed {
				poller := qp.pollers[queueUrl]
				poller.RefreshClient(&maridMetadata.AssumeRoleResult)
			}
			delete(pollerKeys, queueUrl)
		} else {
			queueProvider, err := NewQueueProvider(&maridMetadata)
			if err != nil {
				log.Printf("Poller[%s] could not be added: %s.", queueUrl, err.Error())
				continue
			}
			qp.addPoller(queueProvider).StartPolling()
			log.Printf("Poller[%s] is added.", queueUrl)
		}
	}
	for queueUrl := range pollerKeys {
		qp.removePoller(queueUrl).StopPolling()
		log.Printf("Poller[%s] is removed.", queueUrl)
	}
}

func runQueueProcessor(qp *MaridQueueProcessor) {

	log.Printf("Queue processor has started to run. Refresh client period: %s.", qp.successRefreshPeriod.String())

	ticker := time.NewTicker(qp.successRefreshPeriod)

	for {
		select {
		case <-qp.quit:
			ticker.Stop()
			for queueUrl, poller := range qp.pollers {
				poller.StopPolling()
				delete(qp.pollers, queueUrl)
			}
			log.Printf("Queue processor has stopped to refresh client.")
			return
		case <-ticker.C:
			ticker.Stop()
			token, err := qp.receiveToken()
			if err != nil {
				log.Printf("Refresh cycle of queue processor has failed: %s", err.Error())
				log.Printf("Will refresh token after %s", qp.errorRefreshPeriod.String())
				ticker = time.NewTicker(qp.errorRefreshPeriod)
				break
			}
			qp.successRefreshPeriod = time.Second * time.Duration(token.MaridMetaDataList[0].QueueConfiguration.SuccessRefreshPeriodInSeconds)
			qp.errorRefreshPeriod = time.Second * time.Duration(token.MaridMetaDataList[0].QueueConfiguration.ErrorRefreshPeriodInSeconds)
			qp.refreshPollers(token)

			ticker = time.NewTicker(qp.successRefreshPeriod)
		}
	}
}