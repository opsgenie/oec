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

	setPollingWaitInterval(interval time.Duration) Poller
	setMaxNumberOfMessages(max int64) Poller
	setVisibilityTimeout(timeoutInSeconds int64) Poller
}

type PollerImpl struct {
	queueProvider QueueProvider

	pollingWaitInterval        time.Duration
	maxNumberOfMessages        int64
	visibilityTimeoutInSeconds int64

	state       uint32
	startStopMu *sync.Mutex
	quit        chan struct{}
	wakeUpChan  chan struct{}

	getAvailableWorker      func() uint32 // todo change name
	submit                  func(job Job) (bool, error)
	receiveMessage          func(numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error)
	changeMessageVisibility func(message *sqs.Message, visibilityTimeout int64) error
	isWorkerPoolRunning     func() bool

	releaseMessagesMethod func(p *PollerImpl, messages []*sqs.Message)
	waitMethod            func(p *PollerImpl, pollingPeriod time.Duration)
	runMethod             func(p *PollerImpl)
	wakeUpMethod          func(p *PollerImpl)
	StopPollingMethod     func(p *PollerImpl) error
	StartPollingMethod    func(p *PollerImpl) error
	pollMethod            func(p *PollerImpl) (shouldWait bool)
}

func NewPoller(workerPool WorkerPool, queueProvider QueueProvider, pollingWaitInterval time.Duration, maxNumberOfMessages int64, visibilityTimeoutInSeconds int64) Poller {
	return &PollerImpl{
		quit:                       make(chan struct{}),
		wakeUpChan:                 make(chan struct{}),
		state:                      INITIAL,
		startStopMu:                &sync.Mutex{},
		pollingWaitInterval:        pollingWaitInterval,
		maxNumberOfMessages:        maxNumberOfMessages,
		visibilityTimeoutInSeconds: visibilityTimeoutInSeconds,
		queueProvider:              queueProvider,
		receiveMessage:             queueProvider.ReceiveMessage,
		changeMessageVisibility:    queueProvider.ChangeMessageVisibility,
		getAvailableWorker:         workerPool.GetAvailableWorker,
		submit:                     workerPool.Submit,
		isWorkerPoolRunning:        workerPool.IsRunning,
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
	p.runMethod(p)
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

func (p *PollerImpl) setPollingWaitInterval(interval time.Duration) Poller {
	p.pollingWaitInterval = interval
	return p
}

func (p *PollerImpl) setMaxNumberOfMessages(max int64) Poller {
	p.maxNumberOfMessages = max
	return p
}

func (p *PollerImpl) setVisibilityTimeout(timeoutInSeconds int64) Poller {
	p.visibilityTimeoutInSeconds = timeoutInSeconds
	return p
}

func wakeUpPoller(p *PollerImpl) {

	if atomic.LoadUint32(&p.state) == WAITING {
		p.wakeUpChan <- struct{}{}
	}
}

func StopPolling(p *PollerImpl) error {
	defer p.startStopMu.Unlock()
	p.startStopMu.Lock()

	if atomic.LoadUint32(&p.state) != POLLING {
		return errors.New("Poller is not executing.")
	}

	atomic.StoreUint32(&p.state, FINISHED)

	close(p.quit)
	close(p.wakeUpChan)
	return nil
}

func StartPolling(p *PollerImpl) error {
	defer p.startStopMu.Unlock()
	p.startStopMu.Lock()

	if atomic.LoadUint32(&p.state) != INITIAL {
		return errors.New("Poller is already executing.")
	}

	go p.run()
	atomic.StoreUint32(&p.state, POLLING)

	return nil
}

func releaseMessages(p *PollerImpl, messages []*sqs.Message) {
	for i := 0; i < len(messages); i++ {
		err := p.changeMessageVisibility(messages[i], 0)
		if err != nil {
			// todo
		}

		log.Printf("Message[%s] has been released.", *messages[i].MessageId)
	}
}

func poll(p *PollerImpl) (shouldWait bool) {

	availableWorkerCount := p.getAvailableWorker()
	if availableWorkerCount > 0 {

		maxNumberOfMessages := Min(p.maxNumberOfMessages, int64(availableWorkerCount))
		messages, err := p.receiveMessage(maxNumberOfMessages, p.visibilityTimeoutInSeconds)
		if err != nil { // todo check wait time according to error / check error
			log.Println(err.Error())
			return true
		}

		if len(messages) == 0 {
			log.Println("There is no new message.")
			return true
		}

		for i := 0; i < len(messages); i++ {
			job := NewSqsJob(NewMaridMessage(messages[i]), p.queueProvider, p.visibilityTimeoutInSeconds)
			isSubmitted, err := p.submit(job)
			if err != nil {
				p.releaseMessages(messages[i:])
				return true
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

	log.Println("Will wait " + pollingWaitPeriod.String() + " before next polling")

	for {
		ticker := time.NewTicker(pollingWaitPeriod)
		select {
		case <-p.wakeUpChan:
			log.Println("Sleep interrupted while waiting for next polling.")
			return
		case <-ticker.C:
			return
		}
	}
}

func runPoller(p *PollerImpl) {

	for {
		select {
		case <-p.quit:
			log.Println("Poller has stopped to poll periodically.")
			return
		default:
			if shouldWait := p.poll(); shouldWait {
				p.wait(p.pollingWaitInterval)
			}
		}
	}
}

func Min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}
