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

// Mock
type MockWorkerPool struct {

	IsRunningFunc func() bool
	NumberOfAvailableWorkerFunc func() uint32
	SetKeepAliveTimeFunc func(time.Duration)
	SetMaxNumberOfWorkerFunc func(uint32)
	SetMinNumberOfWorkerFunc func(uint32)
	SetMonitoringPeriodFunc func(time.Duration)
	SetQueueSizeFunc func(uint32)
	StartFunc func() error
	StopFunc func() error
	StopNowFunc func() error
	SubmitFunc func(Job) (bool, error)
}

func NewMockWorkerPool() *MockWorkerPool {
	return &MockWorkerPool{
	}
}

func (m *MockWorkerPool) IsRunning() bool {
	return m.IsRunningFunc()
}

func (m *MockWorkerPool) NumberOfAvailableWorker() uint32 {
	return m.NumberOfAvailableWorkerFunc()
}

func (m *MockWorkerPool) SetKeepAliveTime(duration time.Duration) {
	m.SetKeepAliveTimeFunc(duration)
}

func (m *MockWorkerPool) SetMaxNumberOfWorker(max uint32) {
	m.SetMaxNumberOfWorkerFunc(max)
}

func (m *MockWorkerPool) SetMinNumberOfWorker(min uint32) {
	m.SetMinNumberOfWorkerFunc(min)
}

func (m *MockWorkerPool) SetMonitoringPeriod(duration time.Duration) {
	m.SetMonitoringPeriodFunc(duration)
}

func (m *MockWorkerPool) SetQueueSize(size uint32) {
	m.SetQueueSizeFunc(size)
}

func (m *MockWorkerPool) Start() error {
	return m.StartFunc()
}

func (m *MockWorkerPool) Stop() error {
	return m.StopFunc()
}

func (m *MockWorkerPool) StopNow() error {
	return m.StopNowFunc()
}

func (m *MockWorkerPool) Submit(job Job) (bool, error) {
	return m.SubmitFunc(job)
}