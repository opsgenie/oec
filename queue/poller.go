package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/ois/conf"
	"github.com/opsgenie/ois/git"
	"github.com/opsgenie/ois/util"
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

type OISPoller struct {
	workerPool    WorkerPool
	queueProvider QueueProvider

	integrationId string
	conf          *conf.Configuration
	repositories  git.Repositories

	isRunning          bool
	isRunningWaitGroup *sync.WaitGroup
	startStopMutex     *sync.Mutex
	quit               chan struct{}
	wakeUpChan         chan struct{}
}

func NewPoller(workerPool WorkerPool, queueProvider QueueProvider,
	conf *conf.Configuration, integrationId string,
	repositories git.Repositories) Poller {

	return &OISPoller{
		quit:               make(chan struct{}),
		wakeUpChan:         make(chan struct{}),
		isRunning:          false,
		isRunningWaitGroup: &sync.WaitGroup{},
		startStopMutex:     &sync.Mutex{},
		conf:               conf,
		repositories:       repositories,
		integrationId:      integrationId,
		workerPool:         workerPool,
		queueProvider:      queueProvider,
	}
}

func (p *OISPoller) QueueProvider() QueueProvider {
	return p.queueProvider
}

func (p *OISPoller) RefreshClient(assumeRoleResult AssumeRoleResult) error {
	return p.queueProvider.RefreshClient(assumeRoleResult)
}

func (p *OISPoller) StartPolling() error {
	defer p.startStopMutex.Unlock()
	p.startStopMutex.Lock()

	if p.isRunning {
		return errors.New("Poller is already running.")
	}

	p.isRunningWaitGroup.Add(1)
	go p.run()

	p.isRunning = true

	return nil
}

func (p *OISPoller) StopPolling() error {
	defer p.startStopMutex.Unlock()
	p.startStopMutex.Lock()

	if !p.isRunning {
		return errors.New("Poller is not running.")
	}

	close(p.quit)
	close(p.wakeUpChan)

	p.isRunningWaitGroup.Wait()
	p.isRunning = false

	return nil
}

func (p *OISPoller) terminateMessageVisibility(messages []*sqs.Message) {

	region := p.queueProvider.OISMetadata().Region()

	for i := 0; i < len(messages); i++ {
		messageId := *messages[i].MessageId

		err := p.queueProvider.ChangeMessageVisibility(messages[i], 0)
		if err != nil {
			logrus.Warnf("Poller[%s] could not terminate visibility of message[%s]: %s.", region, messageId, err.Error())
			continue
		}

		logrus.Debugf("Poller[%s] terminated visibility of message[%s].", region, messageId)
	}
}

func (p *OISPoller) poll() (shouldWait bool) {

	availableWorkerCount := p.workerPool.NumberOfAvailableWorker()
	if !(availableWorkerCount > 0) {
		return true
	}

	region := p.queueProvider.OISMetadata().Region()
	maxNumberOfMessages := util.Min(p.conf.PollerConf.MaxNumberOfMessages, int64(availableWorkerCount))

	messages, err := p.queueProvider.ReceiveMessage(maxNumberOfMessages, p.conf.PollerConf.VisibilityTimeoutInSeconds)
	if err != nil { // todo check wait time according to error / check error
		logrus.Errorf("Poller[%s] could not receive message: %s", region, err.Error())
		return true
	}

	messageLength := len(messages)
	if messageLength == 0 {
		logrus.Tracef("There is no new message in the queue[%s].", region)
		return true
	}

	logrus.Debugf("Received %d messages from the queue[%s].", messageLength, region)

	for i := 0; i < messageLength; i++ {

		job := NewSqsJob(
			NewOISMessage(
				messages[i],
				p.repositories,
				&p.conf.ActionSpecifications,
			),
			p.queueProvider,
			p.conf.ApiKey,
			p.conf.BaseUrl,
			p.integrationId,
		)

		isSubmitted, err := p.workerPool.Submit(job)
		if err != nil {
			logrus.Debugf("Error occurred while submitting, messages will be terminated: %s.", err.Error())
			p.terminateMessageVisibility(messages[i:])
			return true
		} else if !isSubmitted {
			p.terminateMessageVisibility(messages[i : i+1])
		}
	}
	return false
}

func (p *OISPoller) wait(pollingWaitInterval time.Duration) {

	queueUrl := p.queueProvider.OISMetadata().QueueUrl()
	logrus.Tracef("Poller[%s] will wait %s before next polling", queueUrl, pollingWaitInterval.String())

	ticker := time.NewTicker(pollingWaitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.wakeUpChan:
			logrus.Debugf("Poller[%s] has been interrupted while waiting for next polling.", queueUrl)
			return
		case <-ticker.C:
			return
		}
	}
}

func (p *OISPoller) run() {

	queueUrl := p.queueProvider.OISMetadata().QueueUrl()
	logrus.Infof("Poller[%s] has started to run.", queueUrl)

	pollingWaitInterval := p.conf.PollerConf.PollingWaitIntervalInMillis * time.Millisecond
	expiredTokenWaitInterval := errorRefreshPeriod

	for {
		select {
		case <-p.quit:
			logrus.Infof("Poller[%s] has stopped to poll.", queueUrl)
			p.isRunningWaitGroup.Done()
			return
		default:
			if p.queueProvider.IsTokenExpired() {
				region := p.queueProvider.OISMetadata().Region()
				logrus.Warnf("Security token is expired, poller[%s] skips to receive message.", region)
				p.wait(expiredTokenWaitInterval)
			} else if shouldWait := p.poll(); shouldWait {
				p.wait(pollingWaitInterval)
			}
		}
	}
}
