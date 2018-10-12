package queue

import (
	"testing"
	"time"
	"github.com/aws/aws-sdk-go/service/sqs"
	"strconv"
	"math/rand"
)

func mockPoll(p *PollerImpl) (shouldWait bool) {
	for j:=0; j< 10 ; j++ {
		for i := 0; i < 10 ; i++ {
			if !p.isWorkerPoolRunning() {
				return
			}
			sqsMessage := &sqs.Message{}
			sqsMessage.SetMessageId(strconv.Itoa(j*10+(i+1)))
			message := NewMaridMessage(sqsMessage)
			job := NewSqsJob(message, p.queueProvider, 1)
			p.submit(job)
		}
		time.Sleep(time.Millisecond * 10)
	}

	return rand.Intn(2) == 0
}

func TestManage(t *testing.T)  {
	rm := NewWorkerPool(
		2,1,10, time.Millisecond * 5, time.Millisecond * 5,
	)

	go rm.Start()
	time.Sleep(time.Second)



	//fmt.Println(rm.GetAvailableWorker())
	time.Sleep(time.Millisecond * 100)
	rm.StopNow()

	time.Sleep(time.Second)
}