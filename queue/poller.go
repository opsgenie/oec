package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Poller interface {
	StartPolling() error
	StopPolling() error

	GetPollingWaitInterval() time.Duration
	GetMaxNumberOfMessages() int64
	GetVisibilityTimeout() int64

	SetPollingWaitInterval(interval *time.Duration) Poller
	SetMaxNumberOfMessages(max *int64) Poller
	SetVisibilityTimeout(timeoutInSeconds *int64) Poller

	GetQueueProvider() QueueProvider
	RefreshClient(assumeRoleResult *AssumeRoleResult) error
}

type PollerImpl struct {
	queueProvider QueueProvider

	pollingWaitInterval        *time.Duration
	maxNumberOfMessages        *int64
	visibilityTimeoutInSeconds *int64

	state       uint32
	startStopMu *sync.Mutex
	quit        chan struct{}
	wakeUpChan  chan struct{}

	getNumberOfAvailableWorker func() uint32
	submit                     func(job Job) (bool, error)
	isWorkerPoolRunning        func() bool

	receiveMessage          func(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)
	changeMessageVisibility func(message *sqs.Message, visibilityTimeout int64) error
	refreshClientMethod   	func(assumeRoleResult *AssumeRoleResult) error
	getQueueUrlMethod	  	func() string

	releaseMessagesMethod func(p *PollerImpl, messages []*sqs.Message)
	waitMethod            func(p *PollerImpl, pollingWaitPeriod time.Duration)
	runMethod             func(p *PollerImpl)
	wakeUpMethod          func(p *PollerImpl)
	StopPollingMethod     func(p *PollerImpl) error
	StartPollingMethod    func(p *PollerImpl) error
	pollMethod            func(p *PollerImpl) (shouldWait bool)
}

func NewPoller(workerPool WorkerPool, queueProvider QueueProvider, pollingWaitInterval *time.Duration, maxNumberOfMessages *int64, visibilityTimeoutInSeconds *int64) Poller {
	return &PollerImpl{
		quit:                       make(chan struct{}),
		wakeUpChan:                 make(chan struct{}),
		state:                      INITIAL,
		startStopMu:                &sync.Mutex{},
		pollingWaitInterval:        pollingWaitInterval,
		maxNumberOfMessages:        maxNumberOfMessages,
		visibilityTimeoutInSeconds: visibilityTimeoutInSeconds,
		getNumberOfAvailableWorker: workerPool.GetNumberOfAvailablefWorker,
		submit:                     workerPool.Submit,
		isWorkerPoolRunning:        workerPool.IsRunning,
		queueProvider:              queueProvider,
		receiveMessage:             queueProvider.ReceiveMessage,
		changeMessageVisibility:    queueProvider.ChangeMessageVisibility,
		refreshClientMethod:        queueProvider.RefreshClient,
		releaseMessagesMethod:      releaseMessages,
		waitMethod:                 waitPolling,
		runMethod:                  runPoller,
		wakeUpMethod:               wakeUpPoller,
		StopPollingMethod:          StopPolling,
		StartPollingMethod:         StartPolling,
		pollMethod:                 poll,
	}
}

func (p *PollerImpl) releaseMessages(messages []*sqs.Message) {
	p.releaseMessagesMethod(p, messages)
}

func (p *PollerImpl) poll() (shouldWait bool) {
	return p.pollMethod(p)
}

func (p *PollerImpl) wait(pollingWaitPeriod time.Duration) {
	p.waitMethod(p, pollingWaitPeriod)
}

func (p *PollerImpl) run() {
	go p.runMethod(p)
}

func (p *PollerImpl) wakeUp() {
	p.wakeUpMethod(p)
}

func (p *PollerImpl) StopPolling() error {
	return p.StopPollingMethod(p)
}

func (p *PollerImpl) StartPolling() error {
	return p.StartPollingMethod(p)
}

func (p *PollerImpl) GetPollingWaitInterval() time.Duration {
	return *p.pollingWaitInterval
}

func (p *PollerImpl) GetMaxNumberOfMessages() int64 {
	return *p.maxNumberOfMessages
}

func (p *PollerImpl) GetVisibilityTimeout() int64 {
	return *p.visibilityTimeoutInSeconds
}

func (p *PollerImpl) SetPollingWaitInterval(interval *time.Duration) Poller {
	p.pollingWaitInterval = interval
	return p
}

func (p *PollerImpl) SetMaxNumberOfMessages(max *int64) Poller {
	p.maxNumberOfMessages = max
	return p
}

func (p *PollerImpl) SetVisibilityTimeout(timeoutInSeconds *int64) Poller {
	p.visibilityTimeoutInSeconds = timeoutInSeconds
	return p
}

func (p *PollerImpl) GetQueueProvider() QueueProvider {
	return p.queueProvider
}


func (p *PollerImpl) RefreshClient(assumeRoleResult *AssumeRoleResult) error {
	return p.refreshClientMethod(assumeRoleResult)
}

func wakeUpPoller(p *PollerImpl) {

	if atomic.LoadUint32(&p.state) == WAITING {
		p.wakeUpChan <- struct{}{}
	}
}

func StopPolling(p *PollerImpl) error {
	defer p.startStopMu.Unlock()
	p.startStopMu.Lock()

	state := atomic.LoadUint32(&p.state)
	if state != POLLING && state != WAITING {
		return errors.New("Poller is not running.")
	}

	close(p.quit)
	close(p.wakeUpChan)

	atomic.StoreUint32(&p.state, FINISHED)

	return nil
}

func StartPolling(p *PollerImpl) error {
	defer p.startStopMu.Unlock()
	p.startStopMu.Lock()

	state := atomic.LoadUint32(&p.state)
	if state != INITIAL /*&& state != FINISHED*/ {
		return errors.New("Poller is already running.")
	}

	p.run()

	atomic.StoreUint32(&p.state, POLLING)

	return nil
}

func releaseMessages(p *PollerImpl, messages []*sqs.Message) {
	for i := 0; i < len(messages); i++ {
		err := p.changeMessageVisibility(messages[i], 0)
		if err != nil {
			log.Printf("Poller[%s] could not release message[%s]: %s.", p.queueProvider.GetMaridMetadata().getQueueUrl() , *messages[i].MessageId, err.Error())
			continue
		}

		log.Printf("Poller[%s] released message[%s].", p.queueProvider.GetMaridMetadata().getQueueUrl() , *messages[i].MessageId)
	}
}

func poll(p *PollerImpl) (shouldWait bool) {

	availableWorkerCount := p.getNumberOfAvailableWorker()
	if availableWorkerCount > 0 {

		maxNumberOfMessages := Min(*p.maxNumberOfMessages, int64(availableWorkerCount))
		messages, err := p.receiveMessage(maxNumberOfMessages, *p.visibilityTimeoutInSeconds)
		if err != nil { // todo check wait time according to error / check error
			log.Println(err.Error())
			return true
		}

		messageLength := len(messages)
		if messageLength == 0 {
			log.Printf("There is no new message in queue[%s].", p.queueProvider.GetMaridMetadata().getQueueUrl())
			return true
		}
		log.Printf("%d messages received.", messageLength)

		for i := 0; i < messageLength; i++ {
			job := NewSqsJob(NewMaridMessage(messages[i]), p.queueProvider, *p.visibilityTimeoutInSeconds)
			isSubmitted, err := p.submit(job)
			if err != nil {
				p.releaseMessages(messages[i:])
				return true	// todo return error or log
			} else if isSubmitted {
				continue
			} else {
				p.releaseMessages(messages[i : i+1])
			}
		}
	}
	return false
}

func waitPolling(p *PollerImpl, pollingWaitPeriod time.Duration) {

	defer atomic.StoreUint32(&p.state, POLLING)
	atomic.StoreUint32(&p.state, WAITING)

	if pollingWaitPeriod == 0 {
		return
	}

	log.Printf("Poller[%s] will wait %s before next polling", p.queueProvider.GetMaridMetadata().getQueueUrl(), pollingWaitPeriod.String())

	for {
		ticker := time.NewTicker(pollingWaitPeriod)
		select {
		case <- p.wakeUpChan:
			ticker.Stop()
			log.Printf("Poller[%s] has been interrupted while waiting for next polling.", p.queueProvider.GetMaridMetadata().getQueueUrl())
			return
		case <- ticker.C:
			return
		}
	}
}

func runPoller(p *PollerImpl) {

	log.Printf("Poller[%s] has started to run.", p.queueProvider.GetMaridMetadata().getQueueUrl())

	for {
		select {
		case <- p.quit:
			log.Printf("Poller[%s] has stopped to poll.", p.queueProvider.GetMaridMetadata().getQueueUrl())
			return
		default:
			if shouldWait := p.poll(); shouldWait {
				p.wait(*p.pollingWaitInterval)
			}
		}
	}
}
