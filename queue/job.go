package queue

import (
	"time"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"sync/atomic"
	"sync"
	"github.com/pkg/errors"
)

const (	// todo change to enum and split
	INITIAL = 0

	EXECUTING = 1
	FINISHED = 2
	ERROR = 3

	POLLING = 4
	WAITING = 5
)

const exceedRetryCount = 3

type Job interface {
	GetJobId() string
	Execute() error
	observe()
}

type SqsJob struct {
	id               *string
	timeoutInSeconds int64
	queueMessage     QueueMessage

	exceedCount 	uint32
	state 			uint32
	mu 				*sync.Mutex
	shouldObserve 	bool
	observer 		*time.Timer
	observePeriod   time.Duration

	changeMessageVisibility 	func(message *sqs.Message, visibilityTimeout int64) (error)
	deleteMessage 				func(message *sqs.Message) (error)

	setStateToExecutingMethod 	func(j *SqsJob) bool
	GetJobIdMethod				func(j *SqsJob) string
	GetMessageMethod 			func(j *SqsJob) *sqs.Message
	ExecuteMethod 				func(j *SqsJob) error
	observeMethod 				func(j *SqsJob)
	checkJobStatusMethod 		func(j *SqsJob)
}

func NewSqsJob(queueMessage QueueMessage, queueProvider QueueProvider, timeoutInSeconds int64) Job {
	return &SqsJob{
		queueMessage:            	queueMessage,
		id:                      	queueMessage.GetMessage().MessageId,
		timeoutInSeconds:        	timeoutInSeconds,
		observePeriod:           	time.Second * time.Duration(timeoutInSeconds - 5),
		state:                   	INITIAL,
		mu:                      	&sync.Mutex{},
		shouldObserve:           	true,
		exceedCount:             	0,
		changeMessageVisibility: 	queueProvider.ChangeMessageVisibility,
		deleteMessage:           	queueProvider.DeleteMessage,
		setStateToExecutingMethod: 	setStateToExecuting,
		GetJobIdMethod: 		   	GetJobId,
		GetMessageMethod: 		   	GetJobMessage,
		ExecuteMethod:  		   	Execute,
		observeMethod: 			   	observe,
		checkJobStatusMethod: 	   	checkJobStatus,
	}
}

func (j *SqsJob) GetJobId() string {
	return j.GetJobIdMethod(j)
}

func (j *SqsJob) GetMessage() *sqs.Message {
	return j.GetMessageMethod(j)
}

func (j *SqsJob) setStateToExecuting() bool {
	return j.setStateToExecutingMethod(j)
}

func (j *SqsJob) Execute() error {
	return j.ExecuteMethod(j)
}

func (j *SqsJob) observe() {
	j.observeMethod(j)
}

func (j *SqsJob) checkJobStatus() {
	j.checkJobStatusMethod(j)
}

func GetJobId(j *SqsJob) string {
	return *j.id
}

func GetJobMessage(j *SqsJob) *sqs.Message {
	return j.queueMessage.GetMessage()
}

func setStateToExecuting(j *SqsJob) bool {
	defer j.mu.Unlock()
	j.mu.Lock()

	if atomic.LoadUint32(&j.state) != INITIAL {
		return false
	}
	atomic.StoreUint32(&j.state, EXECUTING)
	return true
}

func Execute(j *SqsJob) (err error) {
	if !j.setStateToExecuting() {
		return errors.New("Job[" + j.GetJobId() + "] is already executing.")
	}

	defer func() {
		if j.shouldObserve {
			j.observer.Stop()
		}
	}()

	if j.shouldObserve {
		j.observe()
	}

	start := time.Now()
	err = j.queueMessage.Process()
	if err != nil {
		atomic.StoreUint32(&j.state, ERROR)	// todo changeVisibility?
		return err
	}
	took := time.Now().Sub(start)
	log.Printf("Process job[%s] has been done and it took %f seconds.", j.GetJobId(), took.Seconds())

	err = j.deleteMessage(j.GetMessage())
	if err != nil {
		atomic.StoreUint32(&j.state, ERROR) // todo retry delete
		return err
	}
	log.Printf("Message of job[%s] has been deleted.", j.GetJobId())

	atomic.StoreUint32(&j.state, FINISHED)
	return nil
}

func observe(j *SqsJob) {
	state := atomic.LoadUint32(&j.state)
	if state == EXECUTING {
		j.observer = time.AfterFunc(j.observePeriod, j.checkJobStatus)
	}
}

func checkJobStatus(j *SqsJob) {
	state := atomic.LoadUint32(&j.state)
	if state == EXECUTING {
		j.exceedCount++
		if j.exceedCount > exceedRetryCount {
			return
		}
		log.Printf("Timeout is exceed for job[%s] for %d times.", j.GetJobId(), j.exceedCount)
		err := j.changeMessageVisibility(j.queueMessage.GetMessage(), j.timeoutInSeconds)
		if err != nil {
			// todo retry ?
		}
		log.Printf("Message visibility of job[%s] has been changed.", j.GetJobId())
		j.observePeriod = time.Duration(j.timeoutInSeconds) * time.Second
		j.observe()
	}
}

/******************************************************************************************/

type JobQueue struct {
	queue chan Job
	queueSize uint32
	load uint32
}

func NewJobQueue(queueSize uint32) *JobQueue {
	return &JobQueue{
		queue: make(chan Job, queueSize),
		queueSize: queueSize,
		load: 0,
	}
}

func (jq *JobQueue) Increment() {
	atomic.AddUint32(&jq.load, 1)
}

func (jq *JobQueue) Decrement() {
	atomic.AddUint32(&jq.load, ^uint32(0))
}

func (jq *JobQueue) GetQueue() chan Job {
	return jq.queue
}

func (jq *JobQueue) GetLoad() uint32 {
	return atomic.LoadUint32(&jq.load)
}

func (jq *JobQueue) GetLoadFactor() float32 {
	return float32(atomic.LoadUint32(&jq.load) / jq.queueSize)
}

func (jq *JobQueue) IsFull() bool {
	return atomic.LoadUint32(&jq.load) >= jq.queueSize
}

func (jq *JobQueue) Close() {
	close(jq.queue)
}

