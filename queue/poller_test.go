package queue

import (
	"testing"
	"math/rand"
	"sync"
	"github.com/stretchr/testify/assert"
	"time"
	"log"
)

func TestMultipleStartPolling(t *testing.T)  {

	p := &PollerImpl{
		state:       INITIAL,
		startStopMu: &sync.Mutex{},
		getAvailableWorker: func() uint32 {
			return uint32(rand.Int31n(5))
		},
		runMethod: func(p *PollerImpl) {

		},
	}
	err := p.StartPolling()
	assert.Nil(t, err)

	err = p.StartPolling()
	assert.NotNil(t, err)
	assert.Equal(t, "Poller is already executing.", err.Error())
}

func TestStartPolling(t *testing.T)  {

	p := &PollerImpl{
		state:       INITIAL,
		startStopMu: &sync.Mutex{},
		getAvailableWorker: func() uint32 {
			return uint32(rand.Int31n(5))
		},
		runMethod: func(p *PollerImpl) {

		},
	}
	err := p.StartPolling()
	assert.Nil(t, err)

	expectedState := uint32(POLLING)
	assert.Equal(t, expectedState, p.state)
}

func waiting(wakeUp chan struct{}) {

	for {
		ticker := time.NewTicker(time.Second)
		select {
		case <- wakeUp:
			log.Println("Sleep interrupted while waiting for next polling.")
			return
		case <- ticker.C:
			return
		}
	}
}

func TestClose(t *testing.T)  {

	quit := make(chan struct{})
	wakeUp := make(chan struct{})

	go func() {
		for {
			select {
			case <-quit:
				println("quit")
				return
			default:
				waiting(wakeUp)
			}
		}
	}()

	time.Sleep(time.Millisecond * 10)
	close(quit)
	close(wakeUp)

	time.Sleep(time.Second * 2)
}
