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
	RefreshClient(assumeRoleResult AssumeRoleResult) error
}

type MaridPoller struct {
	workerPool    WorkerPool
	queueProvider QueueProvider

	pollingWaitInterval        *time.Duration
	maxNumberOfMessages        *int64
	visibilityTimeoutInSeconds *int64

	state          uint32
	startStopMutex *sync.Mutex
	quit           chan struct{}
	wakeUpChan     chan struct{}

	releaseMessagesMethod func(p *MaridPoller, messages []*sqs.Message)
	waitMethod            func(p *MaridPoller, pollingWaitPeriod time.Duration)
	runMethod             func(p *MaridPoller)
	wakeUpMethod          func(p *MaridPoller)
	StopPollingMethod     func(p *MaridPoller) error
	StartPollingMethod    func(p *MaridPoller) error
	pollMethod            func(p *MaridPoller) (shouldWait bool)
}

func NewPoller(workerPool WorkerPool, queueProvider QueueProvider, pollingWaitInterval *time.Duration, maxNumberOfMessages *int64, visibilityTimeoutInSeconds *int64) Poller {
	return &MaridPoller{
		quit:                       make(chan struct{}),
		wakeUpChan:                 make(chan struct{}),
		state:                      INITIAL,
		startStopMutex:             &sync.Mutex{},
		pollingWaitInterval:        pollingWaitInterval,
		maxNumberOfMessages:        maxNumberOfMessages,
		visibilityTimeoutInSeconds: visibilityTimeoutInSeconds,
		workerPool:                 workerPool,
		queueProvider:              queueProvider,
		releaseMessagesMethod:      releaseMessages,
		waitMethod:                 waitPolling,
		runMethod:                  runPoller,
		wakeUpMethod:               wakeUpPoller,
		StopPollingMethod:          StopPolling,
		StartPollingMethod:         StartPolling,
		pollMethod:                 poll,
	}
}

func (p *MaridPoller) releaseMessages(messages []*sqs.Message) {
	p.releaseMessagesMethod(p, messages)
}

func (p *MaridPoller) poll() (shouldWait bool) {
	return p.pollMethod(p)
}

func (p *MaridPoller) wait(pollingWaitPeriod time.Duration) {
	p.waitMethod(p, pollingWaitPeriod)
}

func (p *MaridPoller) run() {
	go p.runMethod(p)
}

func (p *MaridPoller) wakeUp() {
	p.wakeUpMethod(p)
}

func (p *MaridPoller) StopPolling() error {
	return p.StopPollingMethod(p)
}

func (p *MaridPoller) StartPolling() error {
	return p.StartPollingMethod(p)
}

func (p *MaridPoller) GetPollingWaitInterval() time.Duration {
	return *p.pollingWaitInterval
}

func (p *MaridPoller) GetMaxNumberOfMessages() int64 {
	return *p.maxNumberOfMessages
}

func (p *MaridPoller) GetVisibilityTimeout() int64 {
	return *p.visibilityTimeoutInSeconds
}

func (p *MaridPoller) SetPollingWaitInterval(interval *time.Duration) Poller {
	p.pollingWaitInterval = interval
	return p
}

func (p *MaridPoller) SetMaxNumberOfMessages(max *int64) Poller {
	p.maxNumberOfMessages = max
	return p
}

func (p *MaridPoller) SetVisibilityTimeout(timeoutInSeconds *int64) Poller {
	p.visibilityTimeoutInSeconds = timeoutInSeconds
	return p
}

func (p *MaridPoller) GetQueueProvider() QueueProvider {
	return p.queueProvider
}

func (p *MaridPoller) RefreshClient(assumeRoleResult AssumeRoleResult) error {
	return p.queueProvider.RefreshClient(assumeRoleResult)
}

func wakeUpPoller(p *MaridPoller) {

	if atomic.LoadUint32(&p.state) == WAITING {
		p.wakeUpChan <- struct{}{}
	}
}

func StopPolling(p *MaridPoller) error {
	defer p.startStopMutex.Unlock()
	p.startStopMutex.Lock()

	state := atomic.LoadUint32(&p.state)
	if state != POLLING && state != WAITING {
		return errors.New("Poller is not running.")
	}

	close(p.quit)
	close(p.wakeUpChan)

	atomic.StoreUint32(&p.state, FINISHED)

	return nil
}

func StartPolling(p *MaridPoller) error {
	defer p.startStopMutex.Unlock()
	p.startStopMutex.Lock()

	state := atomic.LoadUint32(&p.state)
	if state != INITIAL /*&& state != FINISHED*/ {
		return errors.New("Poller is already running.")
	}

	p.run()

	atomic.StoreUint32(&p.state, POLLING)

	return nil
}

func releaseMessages(p *MaridPoller, messages []*sqs.Message) {
	for i := 0; i < len(messages); i++ {
		err := p.queueProvider.ChangeMessageVisibility(messages[i], 0)
		if err != nil {
			log.Printf("Poller[%s] could not release message[%s]: %s.", p.queueProvider.GetMaridMetadata().getQueueUrl(), *messages[i].MessageId, err.Error())
			continue
		}

		log.Printf("Poller[%s] released message[%s].", p.queueProvider.GetMaridMetadata().getQueueUrl(), *messages[i].MessageId)
	}
}

func poll(p *MaridPoller) (shouldWait bool) {

	availableWorkerCount := p.workerPool.NumberOfAvailableWorker()
	if availableWorkerCount > 2 {

		maxNumberOfMessages := Min(*p.maxNumberOfMessages, int64(availableWorkerCount))
		messages, err := p.queueProvider.ReceiveMessage(maxNumberOfMessages, *p.visibilityTimeoutInSeconds)
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
			if messages[i].MessageAttributes == nil || *messages[i].MessageAttributes["integrationId"].StringValue != p.queueProvider.GetIntegrationId() {
				p.queueProvider.DeleteMessage(messages[i])
				continue
			}
			job := NewSqsJob(NewMaridMessage(messages[i]), p.queueProvider, *p.visibilityTimeoutInSeconds)
			start := time.Now()
			isSubmitted, err := p.workerPool.Submit(job)
			took := time.Now().Sub(start)
			log.Printf("Submit took %f seconds.", took.Seconds())

			if err != nil {
				p.releaseMessages(messages[i:])
				return true // todo return error or log
			} else if isSubmitted {
				continue
			} else {
				p.releaseMessages(messages[i : i+1])
			}
		}
	}
	return true
}

func waitPolling(p *MaridPoller, pollingWaitPeriod time.Duration) {

	defer atomic.StoreUint32(&p.state, POLLING)
	atomic.StoreUint32(&p.state, WAITING)

	if pollingWaitPeriod == 0 {
		return
	}

	log.Printf("Poller[%s] will wait %s before next polling", p.queueProvider.GetMaridMetadata().getQueueUrl(), pollingWaitPeriod.String())

	for {
		ticker := time.NewTicker(pollingWaitPeriod)
		select {
		case <-p.wakeUpChan:
			ticker.Stop()
			log.Printf("Poller[%s] has been interrupted while waiting for next polling.", p.queueProvider.GetMaridMetadata().getQueueUrl())
			return
		case <-ticker.C:
			return
		}
	}
}

func runPoller(p *MaridPoller) {

	log.Printf("Poller[%s] has started to run.", p.queueProvider.GetMaridMetadata().getQueueUrl())

	for {
		select {
		case <-p.quit:
			log.Printf("Poller[%s] has stopped to poll.", p.queueProvider.GetMaridMetadata().getQueueUrl())
			return
		default:
			if shouldWait := p.poll(); shouldWait {
				p.wait(*p.pollingWaitInterval)
			}
		}
	}
}
