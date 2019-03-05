package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opsgenie/ois/runbook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	JobInitial = iota
	JobExecuting
	JobFinished
	JobError
)

type Job interface {
	JobId() string
	Execute() error
}

type SqsJob struct {
	queueProvider QueueProvider
	queueMessage  QueueMessage

	integrationId string
	apiKey        string
	baseUrl       string

	state        int32
	executeMutex *sync.Mutex
}

func NewSqsJob(queueMessage QueueMessage, queueProvider QueueProvider, apiKey, baseUrl, integrationId string) Job {
	return &SqsJob{
		queueProvider: queueProvider,
		queueMessage:  queueMessage,
		executeMutex:  &sync.Mutex{},
		apiKey:        apiKey,
		baseUrl:       baseUrl,
		integrationId: integrationId,
		state:         JobInitial,
	}
}

func (j *SqsJob) JobId() string {
	return *j.queueMessage.Message().MessageId
}

func (j *SqsJob) SqsMessage() *sqs.Message {
	return j.queueMessage.Message()
}

func (j *SqsJob) Execute() error {

	defer j.executeMutex.Unlock()
	j.executeMutex.Lock()

	if j.state != JobInitial {
		return errors.Errorf("Job[%s] is already executing or finished.", j.JobId())
	}
	j.state = JobExecuting

	region := j.queueProvider.OISMetadata().Region()
	messageId := j.JobId()

	err := j.queueProvider.DeleteMessage(j.SqsMessage())
	if err != nil {
		j.state = JobError
		return errors.Errorf("Message[%s] could not be deleted from the queue[%s]: %s", messageId, region, err)
	}

	logrus.Debugf("Message[%s] is deleted from the queue[%s].", messageId, region)

	messageAttr := j.SqsMessage().MessageAttributes

	if messageAttr == nil ||
		*messageAttr[integrationId].StringValue != j.integrationId {
		j.state = JobError
		return errors.Errorf("Message[%s] is invalid, will not be processed.", messageId)
	}

	result, err := j.queueMessage.Process()
	if err != nil {
		j.state = JobError
		return errors.Errorf("Message[%s] could not be processed: %s", messageId, err)
	}

	go func() {
		start := time.Now()

		err = runbook.SendResultToOpsGenieFunc(result, j.apiKey, j.baseUrl)
		if err != nil {
			logrus.Warnf("Could not send action result[%+v] of message[%s] to Opsgenie: %s", result, messageId, err)
		} else {
			took := time.Since(start)
			logrus.Debugf("Successfully sent result of message[%s] to OpsGenie and it took %f seconds.", messageId, took.Seconds())
		}
	}()

	j.state = JobFinished
	return nil
}
