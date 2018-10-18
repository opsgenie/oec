package queue

import (
	"testing"
)

func TestQueue(t *testing.T) {

	qp := NewQueueProvider().(*MaridQueueProvider)
	qp.refreshClient()
	qp.ReceiveMessage(1, 10)

}
