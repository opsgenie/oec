package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/marid2/conf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Poller interface {
	StartPolling() error
	StopPolling() error

	RefreshClient(assumeRoleResult AssumeRoleResult) error
	QueueProvider() QueueProvider
}

type MaridPoller struct {
	workerPool		WorkerPool
	queueProvider 	QueueProvider

	apiKey 			*string
	baseUrl 		*string
	pollerConf 		*conf.PollerConf
	actionMappings 	*conf.ActionMappings

	isRunning		bool
	startStopMutex 	*sync.Mutex
	quit           	chan struct{}
	wakeUpChan     	chan struct{}
}

func NewPoller(workerPool WorkerPool, queueProvider QueueProvider, pollerConf *conf.PollerConf, actionMappings *conf.ActionMappings, apiKey *string, baseUrl *string) Poller {

	if workerPool == nil || queueProvider == nil || pollerConf == nil || actionMappings == nil || apiKey == nil || baseUrl == nil {
		return nil
	}

	return &MaridPoller {
		quit:           make(chan struct{}),
		wakeUpChan:     make(chan struct{}),
		isRunning:		false,
		startStopMutex: &sync.Mutex{},
		pollerConf:     pollerConf,
		actionMappings: actionMappings,
		apiKey:			apiKey,
		baseUrl:		baseUrl,
		workerPool:     workerPool,
		queueProvider:  queueProvider,
	}
}

func (p *MaridPoller) QueueProvider() QueueProvider {
	return p.queueProvider
}

func (p *MaridPoller) RefreshClient(assumeRoleResult AssumeRoleResult) error {
	return p.queueProvider.RefreshClient(assumeRoleResult)
}

func (p *MaridPoller) StopPolling() error {
	defer p.startStopMutex.Unlock()
	p.startStopMutex.Lock()

	if !p.isRunning {
		return errors.New("Poller is not running.")
	}

	close(p.quit)
	close(p.wakeUpChan)

	p.isRunning = false

	return nil
}

func (p *MaridPoller) StartPolling() error {
	defer p.startStopMutex.Unlock()
	p.startStopMutex.Lock()

	if p.isRunning {
		return errors.New("Poller is already running.")
	}

	go p.run()

	p.isRunning = true

	return nil
}

func (p *MaridPoller) releaseMessages(messages []*sqs.Message) {
	for i := 0; i < len(messages); i++ {
		err := p.queueProvider.ChangeMessageVisibility(messages[i], 0)
		if err != nil {
			logrus.Warnf("Poller[%s] could not release message[%s]: %s.", p.queueProvider.MaridMetadata().QueueUrl() , *messages[i].MessageId, err.Error())
			continue
		}

		logrus.Debugf("Poller[%s] released message[%s].", p.queueProvider.MaridMetadata().QueueUrl() , *messages[i].MessageId)
	}
}

func (p *MaridPoller) poll() (shouldWait bool) {

	availableWorkerCount := p.workerPool.NumberOfAvailableWorker()
	if !(availableWorkerCount > 0) {
		return true
	}

	maxNumberOfMessages := Min(p.pollerConf.MaxNumberOfMessages, int64(availableWorkerCount))
	messages, err := p.queueProvider.ReceiveMessage(maxNumberOfMessages, p.pollerConf.VisibilityTimeoutInSeconds)
	if err != nil { // todo check wait time according to error / check error
		logrus.Println(err.Error())
		return true
	}

	messageLength := len(messages)
	if messageLength == 0 {
		return true
	}

	for i := 0; i < messageLength; i++ {

		if  messages[i].MessageAttributes == nil || *messages[i].MessageAttributes["integrationId"].StringValue != p.queueProvider.IntegrationId() {
			logrus.Debugf("Message[%p] is invalid, will be deleted.", messages[i].MessageId)
			p.queueProvider.DeleteMessage(messages[i])
			continue
		}
		job := NewSqsJob(
			NewMaridMessage(
				messages[i],
				p.actionMappings,
				p.apiKey,
				p.baseUrl,
			),
			p.queueProvider,
		)

		isSubmitted, err := p.workerPool.Submit(job)
		if err != nil {
			logrus.Debugf("Error occurred while submitting: %s", err.Error())
			p.releaseMessages(messages[i:])
			return true
		} else if isSubmitted {
			continue
		} else {
			p.releaseMessages(messages[i : i+1])
		}
	}
	return false
}

func (p *MaridPoller) wait(pollingWaitPeriod time.Duration) {

	if pollingWaitPeriod == 0 {
		return
	}

	logrus.Tracef("Poller[%s] will wait %s before next polling", p.queueProvider.MaridMetadata().QueueUrl(), pollingWaitPeriod.String())

	ticker := time.NewTicker(pollingWaitPeriod)
	defer ticker.Stop()

	for {
		select {
		case <- p.wakeUpChan:
			logrus.Infof("Poller[%s] has been interrupted while waiting for next polling.", p.queueProvider.MaridMetadata().QueueUrl())
			return
		case <- ticker.C:
			return
		}
	}
}

func (p *MaridPoller) run() {

	logrus.Infof("Poller[%s] has started to run.", p.queueProvider.MaridMetadata().QueueUrl())

	pollingWaitInterval := p.pollerConf.PollingWaitIntervalInMillis * time.Millisecond

	for {
		select {
		case <- p.quit:
			logrus.Infof("Poller[%s] has stopped to poll.", p.queueProvider.MaridMetadata().QueueUrl())
			return
		default:
			if shouldWait := p.poll(); shouldWait {
				p.wait(pollingWaitInterval)
			}
		}
	}
}
