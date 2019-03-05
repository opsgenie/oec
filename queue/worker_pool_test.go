package queue

import (
	"github.com/opsgenie/oec/conf"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/cmplx"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

var mockPoolConf = &conf.PoolConf{
	MaxNumberOfWorker:        16,
	MinNumberOfWorker:        2,
	QueueSize:                queueSize,
	KeepAliveTimeInMillis:    keepAliveTimeInMillis,
	MonitoringPeriodInMillis: monitoringPeriodInMillis,
}

var dummyJob = func() {
	var dummy complex128 = 17
	for j := 0; j < 100000; j++ {
		dummy = cmplx.Sin(dummy) + cmplx.Sinh(dummy)
		dummy = cmplx.Acos(dummy) + cmplx.Atanh(dummy)
		dummy = cmplx.Atanh(dummy) + cmplx.Sin(dummy)
		dummy = cmplx.Conj(dummy) - cmplx.Acos(dummy)
		dummy = cmplx.Sinh(dummy) - cmplx.Conj(dummy)
	}
	return
}

func TestMain(m *testing.M) {
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestValidateNewWorkerPool(t *testing.T) {
	configuration := &conf.PoolConf{
		-1,
		-1,
		-1,
		-1,
		-1,
	}
	pool := NewWorkerPool(configuration).(*WorkerPoolImpl)

	assert.Equal(t, int32(minNumberOfWorker), pool.poolConf.MinNumberOfWorker)
	assert.Equal(t, int32(maxNumberOfWorker), pool.poolConf.MaxNumberOfWorker)
	assert.Equal(t, int32(queueSize), pool.poolConf.QueueSize)
	assert.Equal(t, time.Duration(keepAliveTimeInMillis), pool.poolConf.KeepAliveTimeInMillis)
	assert.Equal(t, time.Duration(monitoringPeriodInMillis), pool.poolConf.MonitoringPeriodInMillis)
}

func TestValidateWorkerNumbersNewWorkerPool(t *testing.T) {
	configuration := &conf.PoolConf{
		1,
		2,
		-1,
		0,
		0,
	}
	pool := NewWorkerPool(configuration).(*WorkerPoolImpl)

	assert.Equal(t, int32(1), pool.poolConf.MinNumberOfWorker)
	assert.Equal(t, int32(1), pool.poolConf.MaxNumberOfWorker)
	assert.Equal(t, int32(queueSize), pool.poolConf.QueueSize)
	assert.Equal(t, time.Duration(keepAliveTimeInMillis), pool.poolConf.KeepAliveTimeInMillis)
	assert.Equal(t, time.Duration(monitoringPeriodInMillis), pool.poolConf.MonitoringPeriodInMillis)
}

func TestStartPool(t *testing.T) {

	pool := NewWorkerPool(mockPoolConf).(*WorkerPoolImpl)

	err := pool.Start()

	assert.Nil(t, err)
	assert.Equal(t, 2, int(pool.numberOfCurrentWorker))

	var executeJobCallCount int32 = 0

	for i := 0; i < 1000; i++ {
		job := NewMockJob()
		id := strconv.Itoa(i)
		job.JobIdFunc = func() string {
			return id
		}
		job.ExecuteFunc = func() error {
			atomic.AddInt32(&executeJobCallCount, 1)
			time.Sleep(time.Nanosecond)
			return nil
		}

		for isSubmitted, _ := pool.Submit(job); !isSubmitted; isSubmitted, _ = pool.Submit(job) {
		}
	}

	err = pool.Stop()

	assert.Nil(t, err)
	assert.Equal(t, int32(1000), executeJobCallCount)
}

func BenchmarkWorkerPool(b *testing.B) {

	jobSize1 := 500
	jobSize2 := 1000

	sizes := []struct {
		workerSize int
		jobSize    int
	}{
		{2, jobSize1},
		{2, jobSize2},
		{4, jobSize1},
		{4, jobSize2},
		{8, jobSize1},
		{8, jobSize2},
		{16, jobSize1},
		{16, jobSize2},
		{32, jobSize1},
		{32, jobSize2},
		{64, jobSize1},
		{64, jobSize2},
	}

	for _, size := range sizes {

		pool := NewWorkerPool(
			&conf.PoolConf{
				int32(size.workerSize),
				2,
				queueSize,
				keepAliveTimeInMillis,
				monitoringPeriodInMillis,
			},
		)

		b.Run(strconv.Itoa(size.workerSize)+"MaxWorkers"+strconv.Itoa(size.jobSize)+"Jobs", func(b *testing.B) {

			err := pool.Start()

			assert.Nil(b, err)

			var executeJobCallCount int32 = 0

			for i := 0; i < size.jobSize; i++ {
				job := NewMockJob()
				job.ExecuteFunc = func() error {
					atomic.AddInt32(&executeJobCallCount, 1)
					dummyJob()
					return nil
				}

				for isSubmitted, _ := pool.Submit(job); !isSubmitted; isSubmitted, _ = pool.Submit(job) {
					//time.Sleep(time.Nanosecond)
				}
			}

			err = pool.Stop()

			assert.Nil(b, err)
			assert.Equal(b, int32(size.jobSize), executeJobCallCount)
		})
	}
}

func BenchmarkDummyJob(b *testing.B) {
	dummyJob()
}

func BenchmarkWorkerPoolWithComparableFixedWorkerSize(b *testing.B) {

	jobSize := 500

	cases := []struct {
		maxNumberOfWorker int
		fixed             bool
	}{
		{4, false},
		{4, true},
		{8, false},
		{8, true},
		{16, false},
		{16, true},
		{32, false},
		{32, true},
	}

	for _, testCase := range cases {
		minNumberOfWorker := 2
		maxWorkers := "MaxWorkers"
		if testCase.fixed {
			minNumberOfWorker = testCase.maxNumberOfWorker
			maxWorkers = "FixedWorkers"
		}

		pool := NewWorkerPool(
			&conf.PoolConf{
				int32(testCase.maxNumberOfWorker),
				int32(minNumberOfWorker),
				queueSize,
				keepAliveTimeInMillis,
				monitoringPeriodInMillis,
			},
		)

		b.Run(strconv.Itoa(testCase.maxNumberOfWorker)+maxWorkers+strconv.Itoa(jobSize)+"Jobs", func(b *testing.B) {

			err := pool.Start()

			assert.Nil(b, err)

			var executeJobCallCount int32 = 0

			for i := 0; i < jobSize; i++ {
				job := NewMockJob()
				job.ExecuteFunc = func() error {
					atomic.AddInt32(&executeJobCallCount, 1)
					dummyJob()
					return nil
				}

				for isSubmitted, _ := pool.Submit(job); !isSubmitted; isSubmitted, _ = pool.Submit(job) {
				}
			}

			err = pool.Stop()

			assert.Nil(b, err)
			assert.Equal(b, int32(jobSize), executeJobCallCount)
		})
	}
}

// Mock
type MockWorkerPool struct {
	IsRunningFunc               func() bool
	NumberOfAvailableWorkerFunc func() int32
	StartFunc                   func() error
	StopFunc                    func() error
	StopNowFunc                 func() error
	SubmitFunc                  func(Job) (bool, error)
	SubmitChannelFunc           func() chan<- Job
}

func NewMockWorkerPool() *MockWorkerPool {
	return &MockWorkerPool{}
}

func (m *MockWorkerPool) IsRunning() bool {
	if m.IsRunningFunc != nil {
		return m.IsRunningFunc()
	}
	return false
}

func (m *MockWorkerPool) NumberOfAvailableWorker() int32 {
	if m.NumberOfAvailableWorkerFunc != nil {
		return m.NumberOfAvailableWorkerFunc()
	}
	return 0
}

func (m *MockWorkerPool) Start() error {
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

func (m *MockWorkerPool) Stop() error {
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

func (m *MockWorkerPool) StopNow() error {
	if m.StopNowFunc != nil {
		return m.StopNowFunc()
	}
	return nil
}

func (m *MockWorkerPool) Submit(job Job) (bool, error) {
	if m.SubmitFunc != nil {
		return m.SubmitFunc(job)
	}
	return false, nil
}

func (m *MockWorkerPool) SubmitChannel() chan<- Job {
	if m.SubmitChannelFunc != nil {
		return m.SubmitChannelFunc()
	}
	return nil
}
