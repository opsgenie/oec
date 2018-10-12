package queue

import (
	"sync"
	"github.com/google/uuid"
	"time"
	"log"
	"sync/atomic"
	"github.com/pkg/errors"
)

type WorkerPool interface {
	GetAvailableWorker() 	uint32
	GetIdleWorker() 		uint32
	GetCurrentWorker() 		uint32
	GetMaxWorker() 			uint32
	GetMinWorker() 			uint32

	SetMaxWorker(max uint32)
	SetMinWorker(max uint32)
	SetQueueSize(max uint32)
	SetKeepAliveTime(keepAliveTime time.Duration)
	SetMonitoringPeriod(monitoringPeriod time.Duration)

	Submit(job Job) (bool, error)
	Start()		error
	Stop()		error
	StopNow()	error
	IsRunning() bool
}

func (wp *WorkerPoolImpl) GetMaxWorker() uint32 {
	return atomic.LoadUint32(&wp.maxWorker)
}

func (wp *WorkerPoolImpl) GetMinWorker() uint32 {
	return atomic.LoadUint32(&wp.minWorker)
}

func (wp *WorkerPoolImpl) SetMaxWorker(max uint32) {
	atomic.StoreUint32(&wp.maxWorker, max)
}

func (wp *WorkerPoolImpl) SetMinWorker(max uint32) {
	atomic.StoreUint32(&wp.minWorker, max)
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

func (wp *WorkerPoolImpl) GetAvailableWorker() uint32 {
	return wp.maxWorker - wp.GetCurrentWorker()
}

func (wp *WorkerPoolImpl) GetIdleWorker() uint32 {
	return atomic.LoadUint32(&wp.idleWorker)
}

func (wp *WorkerPoolImpl) GetCurrentWorker() uint32 {
	return atomic.LoadUint32(&wp.currentWorker)
}

type WorkerPoolImpl struct {
	maxWorker 		uint32
	minWorker 		uint32
	currentWorker 	uint32
	activeWorker 	uint32
	idleWorker 		uint32

	keepAliveTime 		time.Duration
	monitoringPeriod 	time.Duration

	jobQueue *JobQueue
	quit     chan struct{}
	stopNow  chan struct{}

	wg          *sync.WaitGroup
	mu          *sync.Mutex
	startStopMu *sync.Mutex

	isRunning 			atomic.Value
	isPrintingMetrics 	atomic.Value // todo can remove
}

func NewWorkerPool(maxWorker uint32, minWorker uint32, queueSize uint32, keepAliveTime time.Duration, monitoringPeriod time.Duration) WorkerPool {
	jobQueue := NewJobQueue(queueSize)

	wp := &WorkerPoolImpl{
		quit:          		make(chan struct{}),
		stopNow:       		make(chan struct{}),
		jobQueue:      		jobQueue,
		maxWorker:     		maxWorker,
		minWorker:     		minWorker,
		wg:            		&sync.WaitGroup{},
		mu:            		&sync.Mutex{},
		startStopMu:   		&sync.Mutex{},
		keepAliveTime: 		keepAliveTime,
		monitoringPeriod: 	monitoringPeriod,
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

	wp.isRunning.Store(true)
	log.Println("Worker pool has started to run.")

	wp.addWorker(wp.minWorker)

	go wp.monitorMetrics(monitoringPeriod)
	go wp.run()

	return nil
}

func (wp *WorkerPoolImpl) IsRunning() bool {

	if wp.isRunning.Load().(bool) {
		return true
	}
	return false
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

	log.Printf("Worker pool is running with; Min Worker: %d, Max Worker: %d", wp.minWorker, wp.maxWorker)

	ticker := time.NewTicker(period)

	for {
		select {
		case <-ticker.C:
			log.Printf("Idle workers: %d, Active workes: %d, Queue load: %d", wp.idleWorker, wp.activeWorker, wp.jobQueue.load)
		case <- wp.quit:
			log.Println("Monitor metrics has stopped.")
			return
		}
	}
}

func (wp *WorkerPoolImpl) addWorker(num uint32) {

	for i := uint32(0); i < num && wp.GetAvailableWorker() > 0; i++ {
		if !wp.IsRunning() {
			return
		}
		atomic.AddUint32(&wp.currentWorker, 1)
		worker := NewWorker(wp)
		go worker.work()
	}
}

func (wp *WorkerPoolImpl) Submit(job Job) (isSubmitted bool, err error) {
	if !wp.IsRunning() {
		log.Println("Worker pool does not accept job now.")
		return false, errors.New("Worker pool is not working")
	}

	if wp.GetIdleWorker() > 0 {
		isSubmitted = true
	} else if wp.GetAvailableWorker() > 0 {
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
		wp.jobQueue.GetQueue() <- job
		wp.jobQueue.Increment()
		log.Printf("Job[%s] is submitted", job.GetJobId())
	} else {
		log.Printf("Job[%s] cannot be submitted", job.GetJobId())
	}

	return isSubmitted, nil
}

func (wp *WorkerPoolImpl) run() {

	for {
		select {
		case <- wp.quit:
			log.Println("Worker pool is waiting to quit.")
			wp.jobQueue.Close()
			wp.wg.Wait()
			log.Println("Worker pool has stopped.")
			return
		case <- wp.stopNow:
			log.Println("Worker pool has stopped immediately.")
			return
		}
	}
}

/******************************************************************************************/

type Worker struct {
	id uuid.UUID
	routineManager *WorkerPoolImpl
}

func NewWorker(routineManager *WorkerPoolImpl) Worker {
	return Worker{
		routineManager:   routineManager,
		id: 			  uuid.New(),
	}
}

func (w *Worker) work() {

	log.Printf("Worker[%s] is spawned.", w.id.String())

	defer w.routineManager.wg.Done()
	w.routineManager.wg.Add(1)

	defer atomic.AddUint32(&w.routineManager.idleWorker, ^uint32(0))
	atomic.AddUint32(&w.routineManager.idleWorker, 1)

	ticker := time.NewTicker(w.routineManager.keepAliveTime)

	if w.routineManager.minWorker == w.routineManager.maxWorker {
		ticker.Stop()
	}

	for {
		select {
		case <- w.routineManager.stopNow:
			log.Printf("Worker [%s] has stopped working.", w.id.String())
			return
		case job, isOpen := <- w.routineManager.jobQueue.GetQueue():
			if !isOpen {
				w.routineManager.decrementCurrentWorker()
				log.Printf("Worker[%s] has done its job.", w.id.String())
				return
			}
			ticker.Stop()
			w.routineManager.jobQueue.Decrement()

			atomic.AddUint32(&w.routineManager.idleWorker, ^uint32(0))
			atomic.AddUint32(&w.routineManager.activeWorker, 1)
			log.Printf("Job[%s] is being processed by Worker[%s].", job.GetJobId(), w.id.String())
			job.Execute()
			atomic.AddUint32(&w.routineManager.activeWorker, ^uint32(0))
			atomic.AddUint32(&w.routineManager.idleWorker, 1)
			log.Printf("Job[%s] has been processed by Worker[%s].", job.GetJobId(), w.id.String())

			ticker = time.NewTicker(w.routineManager.keepAliveTime)
		case <- ticker.C:
			if w.routineManager.checkAndDecrementCurrentWorker() {
				log.Println("Worker [" + w.id.String() + "] has killed itself.")
				return
			}
			ticker = time.NewTicker(w.routineManager.keepAliveTime)
		}
	}
}

func (wp *WorkerPoolImpl) decrementCurrentWorker() {

	defer wp.mu.Unlock()
	wp.mu.Lock()

	atomic.AddUint32(&wp.currentWorker, ^uint32(0))
}

func (wp *WorkerPoolImpl) checkAndDecrementCurrentWorker() bool {	// todo check and remove atomic usage

	defer wp.mu.Unlock()
	wp.mu.Lock()

	if wp.GetCurrentWorker() > wp.minWorker {
		atomic.AddUint32(&wp.currentWorker, ^uint32(0))
		return true
	}
	return false
}