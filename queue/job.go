package queue

import (
	"time"
	"github.com/aws/aws-sdk-go/service/sqs"
	"sync/atomic"
	"sync"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	JobInitial   = 0
	JobExecuting = 1
	JobFinished  = 2
	JobError     = 3
)

const observationExceedRetryCount = 3

type Job interface {
	JobId() string
	Execute() error
}

type SqsJob struct {
	queueProvider QueueProvider

	id               *string
	queueMessage     QueueMessage
	timeoutInSeconds int64

	state         int32
	executeMutex  *sync.Mutex

	shouldObserve          bool
	observationExceedCount int32
	observer               *time.Timer
	observePeriod          time.Duration
}

func NewSqsJob(queueMessage QueueMessage, queueProvider QueueProvider, timeoutInSeconds int64) Job {
	return &SqsJob{
		queueProvider:          queueProvider,
		queueMessage:           queueMessage,
		id:                     queueMessage.Message().MessageId,
		executeMutex:           &sync.Mutex{},
		state:                  JobInitial,
		timeoutInSeconds:       timeoutInSeconds,
		observePeriod:          time.Second * time.Duration(timeoutInSeconds - 5),
		shouldObserve:          false,
		observationExceedCount: 0,
	}
}

func (j *SqsJob) JobId() string {
	return *j.id
}

func (j *SqsJob) JobMessage() *sqs.Message {
	return j.queueMessage.Message()
}

func (j *SqsJob) Execute() (err error) {

	defer j.executeMutex.Unlock()
	j.executeMutex.Lock()

	if atomic.LoadInt32(&j.state) != JobInitial {
		return errors.New("Job[" + j.JobId() + "] is already executing.")
	}
	atomic.StoreInt32(&j.state, JobExecuting)

	defer func() {
		if j.shouldObserve {
			j.observer.Stop()
		}
	}()

	if j.shouldObserve {
		j.observe()
	}

	err = j.queueProvider.DeleteMessage(j.JobMessage())
	if err != nil {
		atomic.StoreInt32(&j.state, JobError)
		return err
	}
	logrus.Debugf("Message of job[%s] has been deleted.", j.JobId())

	start := time.Now()
	err = j.queueMessage.Process()
	if err != nil {
		atomic.StoreInt32(&j.state, JobError)
		return err
	}
	took := time.Now().Sub(start)
	logrus.Debugf("Process job[%s] has been done and it took %f seconds.", j.JobId(), took.Seconds())

	atomic.StoreInt32(&j.state, JobFinished)
	return nil
}

func (j *SqsJob) observe() {
	state := atomic.LoadInt32(&j.state)
	if state == JobExecuting {
		j.observer = time.AfterFunc(j.observePeriod, j.checkJobStatus)
	}
}

func (j *SqsJob) checkJobStatus() {
	state := atomic.LoadInt32(&j.state)
	if state == JobExecuting {
		j.observationExceedCount++
		if j.observationExceedCount > observationExceedRetryCount {
			return
		}
		logrus.Warnf("Timeout is exceed for job[%s] for %d times.", j.JobId(), j.observationExceedCount)
		err := j.queueProvider.ChangeMessageVisibility(j.queueMessage.Message(), j.timeoutInSeconds)
		if err != nil {
			// todo retry ?
		}
		logrus.Debugf("Message visibility of job[%s] has been changed.", j.JobId())
		j.observePeriod = time.Duration(j.timeoutInSeconds) * time.Second
		j.observe()
	}
}