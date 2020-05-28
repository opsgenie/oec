package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/oec/conf"
	"github.com/opsgenie/oec/util"
	"github.com/opsgenie/oec/worker_pool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Poller interface {
	Processor
	RefreshClient(assumeRoleResult AssumeRoleResult) error
	QueueProvider() SQSProvider
}

type poller struct {
	workerPool     worker_pool.WorkerPool
	queueProvider  SQSProvider
	messageHandler MessageHandler

	ownerId            string
	conf               *conf.Configuration
	queueMessageLogrus *logrus.Logger

	isRunning   bool
	isRunningWg *sync.WaitGroup
	startStopMu *sync.Mutex
	quit        chan struct{}
	wakeUp      chan struct{}
}

func NewPoller(workerPool worker_pool.WorkerPool,
	queueProvider SQSProvider,
	messageHandler MessageHandler,
	conf *conf.Configuration,
	ownerId string) Poller {

	return &poller{
		workerPool:         workerPool,
		queueProvider:      queueProvider,
		messageHandler:     messageHandler,
		ownerId:            ownerId,
		conf:               conf,
		queueMessageLogrus: newQueueMessageLogrus(queueProvider.Properties().Region()),
		isRunning:          false,
		isRunningWg:        &sync.WaitGroup{},
		startStopMu:        &sync.Mutex{},
		quit:               make(chan struct{}),
		wakeUp:             make(chan struct{}),
	}
}

func (p *poller) QueueProvider() SQSProvider {
	return p.queueProvider
}

func (p *poller) RefreshClient(assumeRoleResult AssumeRoleResult) error {
	return p.queueProvider.RefreshClient(assumeRoleResult)
}

func (p *poller) Start() error {
	defer p.startStopMu.Unlock()
	p.startStopMu.Lock()

	if p.isRunning {
		return errors.New("Poller is already running.")
	}

	p.isRunningWg.Add(1)
	go p.run()

	p.isRunning = true

	return nil
}

func (p *poller) Stop() error {
	defer p.startStopMu.Unlock()
	p.startStopMu.Lock()

	if !p.isRunning {
		return errors.New("Poller is not running.")
	}

	close(p.quit)
	close(p.wakeUp)

	p.isRunningWg.Wait()
	p.isRunning = false

	return nil
}

func (p *poller) terminateMessageVisibility(messages []*sqs.Message) {

	region := p.queueProvider.Properties().Region()

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

func (p *poller) poll() (shouldWait bool) {

	availableWorkerCount := p.workerPool.NumberOfAvailableWorker()
	if !(availableWorkerCount > 0) {
		return true
	}

	region := p.queueProvider.Properties().Region()
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

		p.queueMessageLogrus.
			WithField("messageId", *messages[i].MessageId).
			Info("Message body: ", *messages[i].Body)

		job := newJob(
			p.queueProvider,
			p.messageHandler,
			*messages[i],
			p.conf.ApiKey,
			p.conf.BaseUrl,
			p.ownerId,
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

func (p *poller) wait(pollingWaitInterval time.Duration) {

	queueUrl := p.queueProvider.Properties().Url()
	logrus.Tracef("Poller[%s] will wait %s before next polling", queueUrl, pollingWaitInterval.String())

	ticker := time.NewTicker(pollingWaitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.wakeUp:
			logrus.Debugf("Poller[%s] has been interrupted while waiting for next polling.", queueUrl)
			return
		case <-ticker.C:
			return
		}
	}
}

func (p *poller) run() {

	queueUrl := p.queueProvider.Properties().Url()
	logrus.Infof("Poller[%s] has started to run.", queueUrl)

	pollingWaitInterval := p.conf.PollerConf.PollingWaitIntervalInMillis * time.Millisecond
	expiredTokenWaitInterval := errorRefreshPeriod

	for {
		select {
		case <-p.quit:
			logrus.Infof("Poller[%s] has stopped to poll.", queueUrl)
			p.isRunningWg.Done()
			return
		default:
			if p.queueProvider.IsTokenExpired() {
				region := p.queueProvider.Properties().Region()
				logrus.Warnf("Security token is expired, poller[%s] skips to receive message.", region)
				p.wait(expiredTokenWaitInterval)
			} else if shouldWait := p.poll(); shouldWait {
				p.wait(pollingWaitInterval)
			}
		}
	}
}

func newQueueMessageLogrus(region string) *logrus.Logger {
	logFilePath := filepath.Join("/var", "log", "opsgenie", "oecQueueMessages-"+region+"-"+strconv.Itoa(os.Getpid())+".log")
	queueMessageLogger := &lumberjack.Logger{
		Filename:  logFilePath,
		MaxSize:   3,  // MB
		MaxAge:    10, // Days
		LocalTime: true,
	}

	queueMessageLogrus := logrus.New()
	queueMessageLogrus.SetFormatter(conf.PrepareLogFormat())

	err := queueMessageLogger.Rotate()
	if err != nil {
		logrus.Info("Cannot create log file for queueMessages. Reason: ", err)
	}

	queueMessageLogrus.SetOutput(queueMessageLogger)

	go util.CheckLogFile(queueMessageLogger, time.Second*10)

	return queueMessageLogrus
}
