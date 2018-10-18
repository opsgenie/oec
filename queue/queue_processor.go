package queue

import (
	"github.com/pkg/errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const maxWorker = 50
const minWorker = 15
const queueSize = 300
const keepAliveTime = time.Second
const monitoringPeriod = 10 * time.Second

const pollingWaitInterval = time.Second * 10
const visibilityTimeoutInSeconds = int64(30)
const maxNumberOfMessages = 10

type QueueProcessor interface {
	Start() error
	Stop() error

	Wait()

	IsWorking() bool

	setMaxWorker(max uint32) QueueProcessor
	setMinWorker(max uint32) QueueProcessor
	setQueueSize(queueSize uint32) QueueProcessor
	setKeepAliveTime(keepAliveTime time.Duration) QueueProcessor
	setMonitoringPeriod(monitoringPeriod time.Duration) QueueProcessor

	setPollingWaitInterval(interval time.Duration) QueueProcessor
	setMaxNumberOfMessages(max int64) QueueProcessor
	setMessageVisibilityTimeout(timeoutInSeconds int64) QueueProcessor
}

type MaridQueueProcessor struct {
	queueProvider QueueProvider // todo move to poller
	workerPool    WorkerPool
	poller        Poller

	isWorking   atomic.Value
	startStopMu *sync.Mutex

	Wg *sync.WaitGroup

	StartMethod func(qp *MaridQueueProcessor) error
}

func NewQueueProcessor() QueueProcessor {
	qp := &MaridQueueProcessor{
		startStopMu: &sync.Mutex{},
		StartMethod: Start,
		Wg:          &sync.WaitGroup{},
	}
	qp.isWorking.Store(false)
	qp.queueProvider = NewQueueProvider()
	qp.workerPool = NewWorkerPool(maxWorker, minWorker, queueSize, keepAliveTime, monitoringPeriod)
	qp.poller = NewPoller(qp.workerPool, qp.queueProvider, pollingWaitInterval, maxNumberOfMessages, visibilityTimeoutInSeconds)

	return qp
}

func (mqp *MaridQueueProcessor) Wait() {
	mqp.Wg.Wait()
}

func (qp *MaridQueueProcessor) Start() error {
	return qp.StartMethod(qp)
}

func Start(qp *MaridQueueProcessor) error {
	defer qp.startStopMu.Unlock()
	qp.startStopMu.Lock()

	if qp.isWorking.Load().(bool) {
		return errors.New("Queue processor is already running.")
	}

	if err := qp.queueProvider.StartRefreshing(); err != nil {
		log.Println("Queue processor could not be started: ", err.Error())
		return err
	}
	qp.workerPool.Start()
	qp.poller.StartPolling()
	qp.isWorking.Store(true)
	qp.Wg.Add(1)
	return nil
}

func (qp *MaridQueueProcessor) Stop() error {
	defer qp.startStopMu.Unlock()
	qp.startStopMu.Lock()

	if !qp.isWorking.Load().(bool) {
		return errors.New("Queue processor already is not running.")
	}

	qp.poller.StopPolling()
	qp.workerPool.Stop()
	qp.queueProvider.StopRefreshing()
	qp.isWorking.Store(false)
	qp.Wg.Done()
	return nil
}

func (wp *MaridQueueProcessor) IsWorking() bool {

	if wp.isWorking.Load().(bool) {
		return true
	}
	return false
}

func (qp *MaridQueueProcessor) setMaxWorker(max uint32) QueueProcessor {
	qp.workerPool.SetMaxWorker(max)
	return qp
}

func (qp *MaridQueueProcessor) setMinWorker(min uint32) QueueProcessor {
	qp.workerPool.SetMinWorker(min)
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

func (qp *MaridQueueProcessor) setPollingWaitInterval(interval time.Duration) QueueProcessor {
	qp.poller.setPollingWaitInterval(interval)
	return qp
}

func (qp *MaridQueueProcessor) setMaxNumberOfMessages(max int64) QueueProcessor {
	qp.poller.setMaxNumberOfMessages(max)
	return qp
}

func (qp *MaridQueueProcessor) setMessageVisibilityTimeout(timeoutInSeconds int64) QueueProcessor {
	qp.poller.setVisibilityTimeout(timeoutInSeconds)
	return qp
}
