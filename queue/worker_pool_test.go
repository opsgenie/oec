package queue

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestManage(t *testing.T)  {
	rm := NewWorkerPool(
		2,1,10, time.Millisecond * 5, time.Millisecond * 5,
	)

	err := rm.Start()
	assert.Nil(t, err)
	err = rm.Stop()
	assert.Nil(t, err)
}