package queue

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestSetQueueProcessor(t *testing.T) {

	qp := NewQueueProcessor().
		setMaxNumberOfMessages(10).
		setMaxWorker(10).
		setMinWorker(3).
		setKeepAliveTime(time.Second).
		setMonitoringPeriod(time.Second * 5)

	wp := qp.(*MaridQueueProcessor).workerPool.(*WorkerPoolImpl)

	expectedMaxNumberOfMessages := uint32(10)
	actualMaxNumberOfMessages := wp.maxWorker

	assert.Equal(t, expectedMaxNumberOfMessages, actualMaxNumberOfMessages)

}

func TestStartQueueProcessor(t *testing.T) {

	qp := NewQueueProcessor().(*MaridQueueProcessor)
	qp.queueProvider.(*MaridQueueProvider).retryer.getMethod = mockHttpGet
	qp.queueProvider.(*MaridQueueProvider).ChangeMessageVisibilityMethod = mockSuccessChange
	qp.queueProvider.(*MaridQueueProvider).DeleteMessageMethod = mockSuccessDelete
	/*qp.queueProvider.(*MaridQueueProvider).ReceiveMessageMethod = func(mqp *MaridQueueProvider, numOfMessage int64, visibilityTimeout int64) ([]*sqs.Message, error) {
		messages := make([]*sqs.Message, 0)

		for j:=0; j < 10 ; j++ {
			for i := 0; i < 10 ; i++ {
				sqsMessage := &sqs.Message{}
				sqsMessage.SetMessageId(strconv.Itoa(j*10+(i+1)))
				messages = append(messages, sqsMessage)
			}
			time.Sleep(time.Millisecond * 10)
		}
		return messages, nil
	}*/
	qp.poller.(*PollerImpl).pollMethod = mockPoll


	err := qp.Start()

	assert.Nil(t, err)

	for {
		if qp.isWorking.Load().(bool) {
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}
}

func TestStartAndImmediatelyStopQueueProcessor(t *testing.T) {

	qp := NewQueueProcessor().(*MaridQueueProcessor)
	qp.queueProvider.(*MaridQueueProvider).retryer.getMethod = mockHttpGet
	qp.queueProvider.(*MaridQueueProvider).ChangeMessageVisibilityMethod = mockSuccessChange
	qp.queueProvider.(*MaridQueueProvider).DeleteMessageMethod = mockSuccessDelete
	qp.poller.(*PollerImpl).pollMethod = mockPoll

	err := qp.Start()
	assert.Nil(t, err)
	err = qp.Stop()
	assert.Nil(t, err)

	time.Sleep(time.Millisecond * 100)
}

func TestStartAndStopQueueProcessor(t *testing.T) {

	qp := NewQueueProcessor().(*MaridQueueProcessor)
	qp.queueProvider.(*MaridQueueProvider).retryer.getMethod = mockHttpGet
	qp.queueProvider.(*MaridQueueProvider).ChangeMessageVisibilityMethod = mockSuccessChange
	qp.queueProvider.(*MaridQueueProvider).DeleteMessageMethod = mockSuccessDelete
	qp.poller.(*PollerImpl).pollMethod = mockPoll

	err := qp.Start()
	assert.Nil(t, err)

	time.Sleep(time.Millisecond * 100)

	err = qp.Stop()
	assert.Nil(t, err)

	time.Sleep(time.Millisecond * 10000)
}

func TestStartQueueProcessorInitialError(t *testing.T) {

	qp := NewQueueProcessor().(*MaridQueueProcessor)
	qp.queueProvider.(*MaridQueueProvider).retryer.getMethod = mockHttpGetError

	err := qp.Start()

	assert.NotNil(t, err)
	assert.Equal(t,"Test http error has occurred while getting token." , err.Error())
}

func TestStopQueueProcessorWhileNotRunning(t *testing.T) {

	qp := NewQueueProcessor()

	err := qp.Stop()

	assert.NotNil(t, err)
	assert.Equal(t,"Queue processor already is not running." , err.Error())
}



