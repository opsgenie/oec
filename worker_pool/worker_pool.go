package worker_pool

import (
	"github.com/opsgenie/oec/conf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxNumberOfWorker        = 12
	minNumberOfWorker        = 4
	queueSize                = 0
	keepAliveTimeInMillis    = 6000
	monitoringPeriodInMillis = 15000
)

type WorkerPool interface {
	Start() error
	Stop() error
	Submit(job Job) (bool, error)
	NumberOfAvailableWorker() int32
}

type workerPool struct {
	poolConf *conf.PoolConf

	numberOfCurrentWorker int32
	numberOfIdleWorker    int32

	jobQueue  chan Job
	quit      chan struct{}
	quitNow   chan struct{}
	isRunning bool

	workersWg        *sync.WaitGroup
	startStopMu      *sync.RWMutex
	numberOfWorkerMu *sync.RWMutex
}

func New(poolConf *conf.PoolConf) WorkerPool {

	if poolConf.MaxNumberOfWorker <= 0 {
		logrus.Infof("Max number of workers should be greater than zero, default value[%d] is set.", maxNumberOfWorker)
		poolConf.MaxNumberOfWorker = maxNumberOfWorker
	}

	if poolConf.MinNumberOfWorker < 0 {
		logrus.Infof("Min number of workers cannot be lesser than zero, default value[%d] is set.", minNumberOfWorker)
		poolConf.MinNumberOfWorker = minNumberOfWorker
	}

	if poolConf.MinNumberOfWorker > poolConf.MaxNumberOfWorker {
		logrus.Infof("Min number of workers cannot be greater than max number of workers, min value is decreased current max value[%d].", maxNumberOfWorker)
		poolConf.MinNumberOfWorker = poolConf.MaxNumberOfWorker
	}

	if poolConf.QueueSize < 0 {
		logrus.Infof("Queue size of the pool cannot be lesser than zero, default value[%d] is set.", queueSize)
		poolConf.QueueSize = queueSize
	}

	if poolConf.KeepAliveTimeInMillis <= 0 {
		logrus.Infof("Keep alive time should be greater than zero, default value[%d ms.] is set.", keepAliveTimeInMillis)
		poolConf.KeepAliveTimeInMillis = keepAliveTimeInMillis
	}

	if poolConf.MonitoringPeriodInMillis <= 0 {
		logrus.Infof("Monitoring period of the pool should be greater than zero, default value[%d ms.] is set.", monitoringPeriodInMillis)
		poolConf.MonitoringPeriodInMillis = monitoringPeriodInMillis
	}

	return &workerPool{
		jobQueue:         make(chan Job, poolConf.QueueSize),
		quit:             make(chan struct{}),
		quitNow:          make(chan struct{}),
		poolConf:         poolConf,
		workersWg:        &sync.WaitGroup{},
		startStopMu:      &sync.RWMutex{},
		numberOfWorkerMu: &sync.RWMutex{},
		isRunning:        false,
	}
}

func (wp *workerPool) Start() error {
	defer wp.startStopMu.Unlock()
	wp.startStopMu.Lock()

	if wp.isRunning {
		return errors.New("Worker pool is already running.")
	}

	go wp.run()
	go wp.monitorMetrics(wp.poolConf.MonitoringPeriodInMillis)

	wp.isRunning = true
	wp.addInitialWorkers(wp.poolConf.MinNumberOfWorker)
	return nil
}

func (wp *workerPool) Stop() error {
	defer wp.startStopMu.Unlock()
	wp.startStopMu.Lock()

	if !wp.isRunning {
		return errors.New("Worker pool is not running.")
	}
	wp.isRunning = false

	logrus.Infof("Worker pool is stopping.")
	close(wp.quit)
	wp.workersWg.Wait()
	logrus.Infof("Worker pool has stopped.")

	return nil
}

func (wp *workerPool) Submit(job Job) (isSubmitted bool, err error) {

	defer wp.startStopMu.RUnlock()
	wp.startStopMu.RLock()

	if !wp.isRunning {
		return false, errors.New("Worker pool is not working")
	}

	logrus.Debugf("Job[%s] is being submitted", job.Id())

	select {
	case wp.jobQueue <- job:
		return true, nil
	default:
		if wp.poolConf.MaxNumberOfWorker == wp.poolConf.MinNumberOfWorker {
			return false, nil
		}

		if wp.CompareAndIncrementCurrentWorker() {
			wp.workersWg.Add(1)
			go func() {
				worker := newWorker(wp)
				worker.work(job)
			}()
			return true, nil
		}

		logrus.Debugf("Job[%s] could not be submitted", job.Id())
		return false, nil
	}
}

func (wp *workerPool) monitorMetrics(monitoringPeriodInMillis time.Duration) {
	if monitoringPeriodInMillis == 0 {
		return
	}

	logrus.Infof("Worker pool is running with; Min Worker: %d, Max Worker: %d, Queue Size: %d", wp.poolConf.MinNumberOfWorker, wp.poolConf.MaxNumberOfWorker, cap(wp.jobQueue))

	ticker := time.NewTicker(monitoringPeriodInMillis * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			logrus.Debugf("Current Worker: %d, Idle Worker: %d, Queue Size: %d, Queue load: %d", wp.NumberOfCurrentWorker(), wp.numberOfIdleWorker, cap(wp.jobQueue), len(wp.jobQueue))
		case <-wp.quit:
			ticker.Stop()
			logrus.Infof("Monitor metrics has stopped.")
			return
		}
	}
}

func (wp *workerPool) addInitialWorkers(num int32) {

	wp.AddNumberOfCurrentAndIdleWorker(num)
	wp.workersWg.Add(int(num))

	for i := int32(0); i < num; i++ {
		worker := newWorker(wp)
		go worker.work(nil)
	}
}

func (wp *workerPool) run() {

	logrus.Infof("Worker pool has started to run.")

	for {
		select {
		case <-wp.quit:
			logrus.Infof("Worker pool is waiting that all workers are done.")
			close(wp.jobQueue)
			wp.workersWg.Wait()
			return
		case <-wp.quitNow:
			logrus.Infof("Worker pool has stopped immediately.")
			return
		}
	}
}

func (wp *workerPool) NumberOfAvailableWorker() int32 {
	wp.numberOfWorkerMu.Lock()
	defer wp.numberOfWorkerMu.Unlock()
	return wp.poolConf.MaxNumberOfWorker - wp.numberOfCurrentWorker + wp.numberOfIdleWorker
}

func (wp *workerPool) NumberOfCurrentWorker() int32 {
	wp.numberOfWorkerMu.RLock()
	defer wp.numberOfWorkerMu.RUnlock()
	return atomic.LoadInt32(&wp.numberOfCurrentWorker)
}

func (wp *workerPool) AddNumberOfCurrentAndIdleWorker(num int32) {
	wp.numberOfWorkerMu.Lock()
	defer wp.numberOfWorkerMu.Unlock()
	wp.numberOfCurrentWorker += num
	wp.numberOfIdleWorker += num
}

func (wp *workerPool) NumberOfIdleWorker() int32 {
	wp.numberOfWorkerMu.RLock()
	defer wp.numberOfWorkerMu.RUnlock()
	return atomic.LoadInt32(&wp.numberOfIdleWorker)
}

func (wp *workerPool) AddNumberOfIdleWorker(num int32) {
	wp.numberOfWorkerMu.Lock()
	defer wp.numberOfWorkerMu.Unlock()
	wp.numberOfIdleWorker += num
}

func (wp *workerPool) CompareAndIncrementCurrentWorker() bool {
	wp.numberOfWorkerMu.Lock()
	defer wp.numberOfWorkerMu.Unlock()
	if wp.numberOfCurrentWorker < wp.poolConf.MaxNumberOfWorker {
		wp.numberOfCurrentWorker++
		wp.numberOfIdleWorker++
		return true
	}
	return false
}

func (wp *workerPool) CompareAndDecrementCurrentWorker() bool {
	wp.numberOfWorkerMu.Lock()
	defer wp.numberOfWorkerMu.Unlock()
	if wp.numberOfCurrentWorker > wp.poolConf.MinNumberOfWorker {
		wp.numberOfCurrentWorker--
		wp.numberOfIdleWorker--
		return true
	}
	return false
}