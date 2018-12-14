package queue

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"time"
	"github.com/opsgenie/marid2/conf"
	"github.com/sirupsen/logrus"
)

type WorkerPool interface {
	Start() error
	Stop() error
	StopNow() error
	Submit(job Job) (bool, error)
	IsRunning() bool

	SubmitChannel() chan<- Job
	NumberOfAvailableWorker() int32
}

type WorkerPoolImpl struct {
	poolConf *conf.PoolConf

	numberOfCurrentWorker int32

	jobQueue  chan Job
	quit      chan struct{}
	quitNow   chan struct{}
	isRunning bool

	workersWaitGroup *sync.WaitGroup
	startStopMutex   *sync.RWMutex
}

func NewWorkerPool(poolConf *conf.PoolConf) WorkerPool {


	if poolConf.MaxNumberOfWorker <= 0 {
		logrus.Infof("Max number of workers should be greater than zero, default value[%d] is set.", maxNumberOfWorker)
		poolConf.MaxNumberOfWorker = maxNumberOfWorker
	}

	if poolConf.MinNumberOfWorker < 0 {
		logrus.Infof("Min number of workers cannot be lesser than zero, default value[%d] is set.", keepAliveTimeInMillis)
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

	if poolConf.MonitoringPeriodInMillis < 0 {
		logrus.Infof("Queue size of the pool cannot be lesser than zero, default value[%d ms.] is set.", monitoringPeriodInMillis)
		poolConf.MonitoringPeriodInMillis = monitoringPeriodInMillis
	}

	return &WorkerPoolImpl{
		jobQueue:         make(chan Job, poolConf.QueueSize),
		quit:             make(chan struct{}),
		quitNow:          make(chan struct{}),
		poolConf:         poolConf,
		workersWaitGroup: &sync.WaitGroup{},
		startStopMutex:   &sync.RWMutex{},
		isRunning:        false,
	}
}

func (wp *WorkerPoolImpl) NumberOfAvailableWorker() int32 {
	return wp.poolConf.MaxNumberOfWorker - wp.NumberOfCurrentWorker()
}

func (wp *WorkerPoolImpl) NumberOfCurrentWorker() int32 {
	return atomic.LoadInt32(&wp.numberOfCurrentWorker)
}

func (wp *WorkerPoolImpl) Stop() error {
	defer wp.startStopMutex.Unlock()
	wp.startStopMutex.Lock()

	if !wp.isRunning {
		return errors.New("Worker pool is not running.")
	}
	wp.isRunning = false

	logrus.Infof("Worker pool is stopping.")
	close(wp.quit)
	wp.workersWaitGroup.Wait()
	logrus.Infof("Worker pool has stopped.")

	return nil
}

func (wp *WorkerPoolImpl) StopNow() error {
	defer wp.startStopMutex.Unlock()
	wp.startStopMutex.Lock()

	if !wp.isRunning {
		return errors.New("Worker pool is not running.")
	}

	logrus.Infof("Worker pool is stopping immediately.")
	close(wp.quitNow)

	wp.isRunning = false
	return nil
}

func (wp *WorkerPoolImpl) Start() error {
	defer wp.startStopMutex.Unlock()
	wp.startStopMutex.Lock()

	if wp.isRunning {
		return errors.New("Worker pool is already running.")
	}

	go wp.run()
	go wp.monitorMetrics(wp.poolConf.MonitoringPeriodInMillis)

	wp.isRunning = true
	wp.addWorker(wp.poolConf.MinNumberOfWorker)
	return nil
}

func (wp *WorkerPoolImpl) IsRunning() bool {
	defer wp.startStopMutex.RUnlock()
	wp.startStopMutex.RLock()

	return wp.isRunning
}

func (wp *WorkerPoolImpl) monitorMetrics(monitoringPeriodInMillis time.Duration) {
	if monitoringPeriodInMillis == 0 {
		return
	}

	logrus.Infof("Worker pool is running with; Min Worker: %d, Max Worker: %d, Queue Size: %d", wp.poolConf.MinNumberOfWorker, wp.poolConf.MaxNumberOfWorker, cap(wp.jobQueue))

	ticker := time.NewTicker(monitoringPeriodInMillis * time.Millisecond)

	for {
		select {
		case <- ticker.C:
			logrus.Debugf("Current Worker: %d, Queue Size: %d, Queue load: %d",wp.NumberOfCurrentWorker(), cap(wp.jobQueue), len(wp.jobQueue))
		case <- wp.quit:
			ticker.Stop()
			logrus.Infof("Monitor metrics has stopped.")
			return
		}
	}
}

func (wp *WorkerPoolImpl) addWorker(num int32) {

	for i := int32(0); i < num && wp.NumberOfAvailableWorker() > 0; i++ {
		if !wp.isRunning {
			return
		}
		atomic.AddInt32(&wp.numberOfCurrentWorker, 1)
		wp.workersWaitGroup.Add(1)
		worker := NewWorker(wp)
		go worker.work()
	}
}

func (wp *WorkerPoolImpl) SubmitChannel() chan<- Job {
	return wp.jobQueue
}


func (wp *WorkerPoolImpl) Submit(job Job) (isSubmitted bool, err error) {

	defer wp.startStopMutex.RUnlock()
	wp.startStopMutex.RLock()

	if !wp.isRunning {
		logrus.Debugf("Worker pool does not accept job now.")
		return false, errors.New("Worker pool is not working")
	}

	logrus.Debugf("Job[%s] is being submmitted", job.JobId())

	select {
	case wp.jobQueue <- job:
		return true, nil
	default:
		for {
			if wp.poolConf.MaxNumberOfWorker == wp.poolConf.MinNumberOfWorker {
				return false, nil
			}

			numberOfCurrentWorker := wp.NumberOfCurrentWorker()
			if numberOfCurrentWorker < wp.poolConf.MaxNumberOfWorker {
				if !atomic.CompareAndSwapInt32(&wp.numberOfCurrentWorker, numberOfCurrentWorker, numberOfCurrentWorker+1) {
					continue
				}
				//atomic.AddInt32(&wp.numberOfCurrentWorker, 1)
				wp.workersWaitGroup.Add(1)
				go func() {
					worker := NewWorker(wp)
					worker.doJob(job)
					worker.work()
				}()
				return true, nil
			} else {
				logrus.Debugf("Job[%s] cannot be submitted", job.JobId())
				return false, nil
			}
		}
	}
}

func (wp *WorkerPoolImpl) run() {

	logrus.Infof("Worker pool has started to run.")

	for {
		select {
		case <- wp.quit:
			logrus.Infof("Worker pool is waiting to quit.")
			close(wp.jobQueue)
			wp.workersWaitGroup.Wait()
			return
		case <- wp.quitNow:
			logrus.Infof("Worker pool has stopped immediately.")
			return
		}
	}
}

/******************************************************************************************/

type Worker struct {
	id         uuid.UUID
	workerPool *WorkerPoolImpl
}

func NewWorker(workerPool *WorkerPoolImpl) Worker {
	return Worker{
		workerPool: workerPool,
		id:         uuid.New(),
	}
}

func (w *Worker) doJob(job Job) {
	logrus.Debugf("Job[%s] is submitted", job.JobId())

	err := job.Execute() // todo panic recover, stay the pool as working
	if err != nil {
		logrus.Errorf(err.Error())
	}

	logrus.Debugf("Job[%s] has been processed by Worker[%s].", job.JobId(), w.id.String())
}

func (w *Worker) work() {

	logrus.Debugf("Worker[%s] is spawned.", w.id.String())
	defer w.workerPool.workersWaitGroup.Done()

	ticker := time.NewTicker(w.workerPool.poolConf.KeepAliveTimeInMillis * time.Millisecond)

	if w.workerPool.poolConf.MinNumberOfWorker == w.workerPool.poolConf.MaxNumberOfWorker {
		ticker.Stop()
	}

	defer ticker.Stop()

	for {
		select {
		case <- w.workerPool.quitNow:
			logrus.Debugf("Worker [%s] has stopped working.", w.id.String())
			atomic.AddInt32(&w.workerPool.numberOfCurrentWorker, -1)
			return
		case job, isOpen := <- w.workerPool.jobQueue:
			if !isOpen {
				atomic.AddInt32(&w.workerPool.numberOfCurrentWorker, -1)
				logrus.Debugf("Worker[%s] has done its job.", w.id.String())
				return
			}
			ticker.Stop()

			w.doJob(job)

			ticker = time.NewTicker(w.workerPool.poolConf.KeepAliveTimeInMillis)
		case <- ticker.C:
			ticker.Stop()

			currentNumber := w.workerPool.NumberOfCurrentWorker()
			if currentNumber > w.workerPool.poolConf.MinNumberOfWorker {
				if !atomic.CompareAndSwapInt32(&w.workerPool.numberOfCurrentWorker, currentNumber, currentNumber - 1) {
					break
				}
				logrus.Debugf("Worker [%s] has killed itself.", w.id.String())
				return
			}

			ticker = time.NewTicker(w.workerPool.poolConf.KeepAliveTimeInMillis)

		}
	}
}
