package queue

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type WorkerPool interface {
	Submit(job Job) (bool, error)
	Start() error
	Stop() error
	StopNow() error
	IsRunning() bool

	NumberOfAvailableWorker() uint32

	SetMaxNumberOfWorker(max uint32)
	SetMinNumberOfWorker(max uint32)
	SetQueueSize(max uint32)
	SetKeepAliveTime(keepAliveTime time.Duration)
	SetMonitoringPeriod(monitoringPeriod time.Duration)
}

func (wp *WorkerPoolImpl) SetMaxNumberOfWorker(max uint32) {
	atomic.StoreUint32(&wp.maxNumberOfWorker, max)
}

func (wp *WorkerPoolImpl) SetMinNumberOfWorker(max uint32) {
	atomic.StoreUint32(&wp.minNumberOfWorker, max)
}

func (wp *WorkerPoolImpl) SetQueueSize(max uint32) {
	if !wp.IsRunning() {
		wp.jobQueue = NewJobQueue(max)
	}
}

func (wp *WorkerPoolImpl) SetKeepAliveTime(keepAliveTime time.Duration) {
	wp.keepAliveTime = keepAliveTime
}

func (wp *WorkerPoolImpl) SetMonitoringPeriod(monitoringPeriod time.Duration) {
	wp.monitoringPeriod = monitoringPeriod
}

func (wp *WorkerPoolImpl) NumberOfAvailableWorker() uint32 {
	return wp.maxNumberOfWorker - atomic.LoadUint32(&wp.numberOfActiveWorker)
}

func (wp *WorkerPoolImpl) getNumberOfAddableWorker() uint32 {
	return wp.maxNumberOfWorker - wp.getNumberOfCurrentWorker()
}

func (wp *WorkerPoolImpl) getNumberOfIdleWorker() uint32 {
	return atomic.LoadUint32(&wp.numberOfIdleWorker)
}

func (wp *WorkerPoolImpl) getNumberOfCurrentWorker() uint32 {
	return atomic.LoadUint32(&wp.numberOfCurrentWorker)
}

type WorkerPoolImpl struct {
	maxNumberOfWorker     uint32
	minNumberOfWorker     uint32
	numberOfCurrentWorker uint32
	numberOfActiveWorker  uint32
	numberOfIdleWorker    uint32

	keepAliveTime    time.Duration
	monitoringPeriod time.Duration

	jobQueue *JobQueue
	quit     chan struct{}
	stopNow  chan struct{}

	workersWaitGroup *sync.WaitGroup
	workerCountMutex *sync.Mutex
	startStopMutex   *sync.Mutex

	isRunning bool

	submitFunc func(wp *WorkerPoolImpl, job Job) (isSubmitted bool, err error)
}

func NewWorkerPool(maxWorker uint32, minWorker uint32, queueSize uint32, keepAliveTime time.Duration, monitoringPeriod time.Duration) WorkerPool {

	wp := &WorkerPoolImpl{
		quit:              make(chan struct{}),
		stopNow:           make(chan struct{}),
		jobQueue:          NewJobQueue(queueSize),
		maxNumberOfWorker: maxWorker,
		minNumberOfWorker: minWorker,
		workersWaitGroup:  &sync.WaitGroup{},
		workerCountMutex:  &sync.Mutex{},
		startStopMutex:    &sync.Mutex{},
		keepAliveTime:     keepAliveTime,
		monitoringPeriod:  monitoringPeriod,
		isRunning:         false,
		submitFunc:        Submit,
	}

	return wp
}

func (wp *WorkerPoolImpl) Stop() error {
	wp.startStopMutex.Lock()

	if !wp.isRunning {
		return errors.New("Worker pool is not running.")
	}
	wp.isRunning = false

	wp.startStopMutex.Unlock()

	log.Println("Worker pool is stopping.")
	close(wp.quit)
	return nil
}

func (wp *WorkerPoolImpl) StopNow() error {
	wp.startStopMutex.Lock()

	if !wp.isRunning {
		return errors.New("Worker pool is not running.")
	}
	wp.isRunning = false

	wp.startStopMutex.Unlock()

	log.Println("Worker pool is stopping immediately.")
	close(wp.stopNow)
	return nil
}

func (wp *WorkerPoolImpl) Start() error {
	wp.startStopMutex.Lock()

	if wp.isRunning {
		return errors.New("Worker pool is already running.")
	}
	wp.isRunning = true

	wp.startStopMutex.Unlock()

	go wp.run()
	go wp.monitorMetrics(monitoringPeriod)

	wp.addWorker(wp.minNumberOfWorker)

	return nil
}

func (wp *WorkerPoolImpl) IsRunning() bool {
	defer wp.startStopMutex.Unlock()
	wp.startStopMutex.Lock()

	return wp.isRunning
}

func (wp *WorkerPoolImpl) monitorMetrics(period time.Duration) {
	if period == 0 {
		return
	}

	log.Printf("Worker pool is running with; Min Worker: %d, Max Worker: %d, Queue Size: %d", wp.minNumberOfWorker, wp.maxNumberOfWorker, wp.jobQueue.GetSize())

	ticker := time.NewTicker(period)

	for {
		select {
		case <-ticker.C:
			log.Printf("Idle workers: %d, Active workers: %d, Queue Size: %d, Queue load factor: %%%d", wp.numberOfIdleWorker, wp.numberOfActiveWorker, wp.jobQueue.GetSize(), int(wp.jobQueue.GetLoadFactor()*100))
		case <-wp.quit:
			log.Println("Monitor metrics has stopped.")
			return
		}
	}
}

func (wp *WorkerPoolImpl) addWorker(num uint32) {

	for i := uint32(0); i < num && wp.getNumberOfAddableWorker() > 0; i++ {
		if !wp.isRunning {
			return
		}
		atomic.AddUint32(&wp.numberOfIdleWorker, 1)
		atomic.AddUint32(&wp.numberOfCurrentWorker, 1)
		worker := NewWorker(wp)
		go worker.work()
	}
}

func (wp *WorkerPoolImpl) Submit(job Job) (isSubmitted bool, err error) {
	return wp.submitFunc(wp, job)
}

func Submit(wp *WorkerPoolImpl, job Job) (isSubmitted bool, err error) {
	if !wp.isRunning {
		log.Println("Worker pool does not accept job now.")
		return false, errors.New("Worker pool is not working")
	}

	if wp.getNumberOfIdleWorker() > 2 {
		isSubmitted = true
	} else if wp.getNumberOfAddableWorker() > 0 {
		wp.addWorker(1)
		isSubmitted = true
	} else {
		if !wp.jobQueue.IsFull() {
			isSubmitted = true
		} else {
			isSubmitted = false
		}
	}

	if isSubmitted {
		log.Printf("Job[%s] is being submmitted", job.GetJobId())
		wp.jobQueue.GetChan() <- job // todo check queue channel after closing it
		wp.jobQueue.Increment()
		log.Printf("Job[%s] is submitted", job.GetJobId())
	} else {
		log.Printf("Job[%s] cannot be submitted", job.GetJobId())
	}

	return isSubmitted, nil
}

func (wp *WorkerPoolImpl) run() {

	log.Println("Worker pool has started to run.")

	for {
		select {
		case <-wp.quit:
			log.Println("Worker pool is waiting to quit.")
			wp.jobQueue.Close()
			wp.workersWaitGroup.Wait()
			log.Println("Worker pool has stopped.")
			return
		case <-wp.stopNow:
			log.Println("Worker pool has stopped immediately.")
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

func (w *Worker) work() {

	log.Printf("Worker[%s] is spawned.", w.id.String())

	defer w.workerPool.workersWaitGroup.Done()
	w.workerPool.workersWaitGroup.Add(1)

	defer atomic.AddUint32(&w.workerPool.numberOfIdleWorker, ^uint32(0))

	ticker := time.NewTicker(w.workerPool.keepAliveTime)

	if w.workerPool.minNumberOfWorker == w.workerPool.maxNumberOfWorker {
		ticker.Stop()
	}

	for {
		select {
		case <-w.workerPool.stopNow:
			log.Printf("Worker [%s] has stopped working.", w.id.String())
			return
		case job, isOpen := <-w.workerPool.jobQueue.GetChan():
			if !isOpen {
				atomic.AddUint32(&w.workerPool.numberOfCurrentWorker, ^uint32(0))
				log.Printf("Worker[%s] has done its job.", w.id.String())
				return
			}
			ticker.Stop()
			w.workerPool.jobQueue.Decrement()

			atomic.AddUint32(&w.workerPool.numberOfIdleWorker, ^uint32(0))
			atomic.AddUint32(&w.workerPool.numberOfActiveWorker, 1)
			log.Printf("Job[%s] is being processed by Worker[%s].", job.GetJobId(), w.id.String())
			err := job.Execute()
			if err != nil {
				log.Println(err)
			}
			atomic.AddUint32(&w.workerPool.numberOfActiveWorker, ^uint32(0))
			atomic.AddUint32(&w.workerPool.numberOfIdleWorker, 1)
			log.Printf("Job[%s] has been processed by Worker[%s].", job.GetJobId(), w.id.String())

			ticker = time.NewTicker(w.workerPool.keepAliveTime)
		case <-ticker.C:
			if w.workerPool.checkAndDecrementCurrentWorker() {
				log.Println("Worker [" + w.id.String() + "] has killed itself.")
				return
			}
			ticker = time.NewTicker(w.workerPool.keepAliveTime)
		}
	}
}

func (wp *WorkerPoolImpl) checkAndDecrementCurrentWorker() bool { // todo check and remove atomic usage

	defer wp.workerCountMutex.Unlock()
	wp.workerCountMutex.Lock()

	if wp.getNumberOfCurrentWorker() > wp.minNumberOfWorker {
		atomic.AddUint32(&wp.numberOfCurrentWorker, ^uint32(0))
		return true
	}
	return false
}
