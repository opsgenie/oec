package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	JobInitial   = 0
	JobExecuting = 1
	JobFinished  = 2
	JobError     = 3
)

type Job interface {
	JobId() string
	Execute() error
}

type SqsJob struct {
	queueProvider QueueProvider

	id               *string
	queueMessage     QueueMessage

	state         int32
	executeMutex  *sync.Mutex
}

func NewSqsJob(queueMessage QueueMessage, queueProvider QueueProvider) Job {
	return &SqsJob{
		queueProvider:          queueProvider,
		queueMessage:           queueMessage,
		id:                     queueMessage.Message().MessageId,
		executeMutex:           &sync.Mutex{},
		state:                  JobInitial,
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

	if j.state != JobInitial {
		return errors.New("Job[" + j.JobId() + "] is already executing.")
	}
	j.state = JobExecuting

	err = j.queueProvider.DeleteMessage(j.JobMessage())
	if err != nil {
		j.state = JobError
		return err
	}

	start := time.Now()
	err = j.queueMessage.Process()
	if err != nil {
		j.state = JobError
		return err
	}
	took := time.Now().Sub(start)
	logrus.Debugf("Process job[%s] has been done and it took %f seconds.", j.JobId(), took.Seconds())

	j.state = JobFinished
	return nil
}