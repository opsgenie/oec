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
	GetNumberOfAvailablefWorker() uint32
	GetNumberOfIdleWorker() uint32
	GetNumberOfCurrentWorker() uint32
	GetMaxNumberOfWorker() uint32
	GetMinNumberOfWorker() uint32

	SetMaxNumberOfWorker(max uint32)
	SetMinNumberOfWorker(max uint32)
	SetQueueSize(max uint32)
	SetKeepAliveTime(keepAliveTime time.Duration)
	SetMonitoringPeriod(monitoringPeriod time.Duration)

	Submit(job Job) (bool, error)
	Start() error
	Stop() error
	StopNow() error
	IsRunning() bool
}

func (wp *WorkerPoolImpl) GetMaxNumberOfWorker() uint32 {
	return atomic.LoadUint32(&wp.maxNumberOfWorker)
}

func (wp *WorkerPoolImpl) GetMinNumberOfWorker() uint32 {
	return atomic.LoadUint32(&wp.minNumberOfWorker)
}

func (wp *WorkerPoolImpl) SetMaxNumberOfWorker(max uint32) {
	atomic.StoreUint32(&wp.maxNumberOfWorker, max)
}

func (wp *WorkerPoolImpl) SetMinNumberOfWorker(max uint32) {
	atomic.StoreUint32(&wp.minNumberOfWorker, max)
}

func (wp *WorkerPoolImpl) SetQueueSize(max uint32) {
	//
}

func (wp *WorkerPoolImpl) SetKeepAliveTime(keepAliveTime time.Duration) {
	wp.keepAliveTime = keepAliveTime
}

func (wp *WorkerPoolImpl) SetMonitoringPeriod(monitoringPeriod time.Duration) {
	wp.monitoringPeriod = monitoringPeriod
}

func (wp *WorkerPoolImpl) GetNumberOfAvailablefWorker() uint32 {
	return wp.maxNumberOfWorker - wp.GetNumberOfCurrentWorker()
}

func (wp *WorkerPoolImpl) GetNumberOfIdleWorker() uint32 {
	return atomic.LoadUint32(&wp.numberOfIdleWorker)
}

func (wp *WorkerPoolImpl) GetNumberOfCurrentWorker() uint32 {
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

	wg          *sync.WaitGroup
	mu          *sync.Mutex
	startStopMu *sync.Mutex

	isRunning         atomic.Value
	isPrintingMetrics atomic.Value // todo can remove
}

func NewWorkerPool(maxWorker uint32, minWorker uint32, queueSize uint32, keepAliveTime time.Duration, monitoringPeriod time.Duration) WorkerPool {
	jobQueue := NewJobQueue(queueSize)

	wp := &WorkerPoolImpl{
		quit:              make(chan struct{}),
		stopNow:           make(chan struct{}),
		jobQueue:          jobQueue,
		maxNumberOfWorker: maxWorker,
		minNumberOfWorker: minWorker,
		wg:                &sync.WaitGroup{},
		mu:                &sync.Mutex{},
		startStopMu:       &sync.Mutex{},
		keepAliveTime:     keepAliveTime,
		monitoringPeriod:  monitoringPeriod,
	}
	wp.isRunning.Store(false)
	wp.isPrintingMetrics.Store(false)

	return wp
}

func (wp *WorkerPoolImpl) Stop() error {
	defer wp.startStopMu.Unlock()
	wp.startStopMu.Lock()

	if !wp.IsRunning() {
		return errors.New("Worker pool is not running.")
	}
	wp.isRunning.Store(false)
	log.Println("Worker pool is stopping.")
	close(wp.quit)
	return nil
}

func (wp *WorkerPoolImpl) StopNow() error {
	defer wp.startStopMu.Unlock()
	wp.startStopMu.Lock()

	if !wp.IsRunning() {
		return errors.New("Worker pool is not running.")
	}
	wp.isRunning.Store(false)
	log.Println("Worker pool is stopping immediately.")
	close(wp.stopNow)
	return nil
}

func (wp *WorkerPoolImpl) Start() error {
	defer wp.startStopMu.Unlock()
	wp.startStopMu.Lock()

	if wp.IsRunning() {
		return errors.New("Worker pool is already running.")
	}

	go wp.run()
	go wp.monitorMetrics(monitoringPeriod)

	wp.isRunning.Store(true)

	wp.addWorker(wp.minNumberOfWorker)

	return nil
}

func (wp *WorkerPoolImpl) IsRunning() bool {
	return wp.isRunning.Load().(bool)
}

func (wp *WorkerPoolImpl) monitorMetrics(period time.Duration) {
	if period == 0 {
		return
	}

	if wp.isPrintingMetrics.Load().(bool) {
		return
	}

	defer wp.isPrintingMetrics.Store(false)
	wp.isPrintingMetrics.Store(true)

	log.Printf("Worker pool is running with; Min Worker: %d, Max Worker: %d, Queue Size: %d", wp.minNumberOfWorker, wp.maxNumberOfWorker, wp.jobQueue.GetSize())

	ticker := time.NewTicker(period)

	for {
		select {
		case <-ticker.C:
			log.Printf("Idle workers: %d, Active workes: %d, Queue Size: %d, Queue load factor: %f", wp.numberOfIdleWorker, wp.numberOfActiveWorker, wp.jobQueue.GetSize(), wp.jobQueue.GetLoadFactor())
		case <-wp.quit:
			log.Println("Monitor metrics has stopped.")
			return
		}
	}
}

func (wp *WorkerPoolImpl) addWorker(num uint32) {

	for i := uint32(0); i < num && wp.GetNumberOfAvailablefWorker() > 0; i++ {
		if !wp.IsRunning() {
			return
		}
		atomic.AddUint32(&wp.numberOfCurrentWorker, 1)
		worker := NewWorker(wp)
		go worker.work()
	}
}

func (wp *WorkerPoolImpl) Submit(job Job) (isSubmitted bool, err error) {
	if !wp.IsRunning() {
		log.Println("Worker pool does not accept job now.")
		return false, errors.New("Worker pool is not working")
	}

	if wp.GetNumberOfIdleWorker() > 0 {
		isSubmitted = true
	} else if wp.GetNumberOfAvailablefWorker() > 0 {
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
		wp.jobQueue.GetQueue() <- job	// todo check queue channel after closing it
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
			wp.wg.Wait()
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

	defer w.workerPool.wg.Done()
	w.workerPool.wg.Add(1)

	defer atomic.AddUint32(&w.workerPool.numberOfIdleWorker, ^uint32(0))
	atomic.AddUint32(&w.workerPool.numberOfIdleWorker, 1)

	ticker := time.NewTicker(w.workerPool.keepAliveTime)

	if w.workerPool.minNumberOfWorker == w.workerPool.maxNumberOfWorker {
		ticker.Stop()
	}

	for {
		select {
		case <-w.workerPool.stopNow:
			log.Printf("Worker [%s] has stopped working.", w.id.String())
			return
		case job, isOpen := <-w.workerPool.jobQueue.GetQueue():
			if !isOpen {
				w.workerPool.decrementCurrentWorker()
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

func (wp *WorkerPoolImpl) decrementCurrentWorker() {

	defer wp.mu.Unlock()
	wp.mu.Lock()

	atomic.AddUint32(&wp.numberOfCurrentWorker, ^uint32(0))
}

func (wp *WorkerPoolImpl) checkAndDecrementCurrentWorker() bool { // todo check and remove atomic usage

	defer wp.mu.Unlock()
	wp.mu.Lock()

	if wp.GetNumberOfCurrentWorker() > wp.minNumberOfWorker {
		atomic.AddUint32(&wp.numberOfCurrentWorker, ^uint32(0))
		return true
	}
	return false
}
