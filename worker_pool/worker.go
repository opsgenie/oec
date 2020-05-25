package worker_pool

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

type worker struct {
	id         uuid.UUID
	workerPool *workerPool
}

func newWorker(workerPool *workerPool) worker {
	return worker{
		workerPool: workerPool,
		id:         uuid.New(),
	}
}

func (w *worker) doJob(job Job) {
	defer w.workerPool.AddNumberOfIdleWorker(1)
	w.workerPool.AddNumberOfIdleWorker(-1)

	logrus.Debugf("Job[%s] is submitted to worker[%s]", job.Id(), w.id.String())

	err := job.Execute() // todo panic recover, stay the pool as working
	if err != nil {
		logrus.Errorf(err.Error())
		return
	}

	logrus.Debugf("Job[%s] has been processed by worker[%s].", job.Id(), w.id.String())
}

func (w *worker) work(initialJob Job) {

	logrus.Debugf("worker[%s] is spawned.", w.id.String())
	defer w.workerPool.workersWg.Done()

	if initialJob != nil {
		w.doJob(initialJob)
	}

	if w.workerPool.poolConf.MinNumberOfWorker == w.workerPool.poolConf.MaxNumberOfWorker {
		w.runWithFixedNumberOfWorker()
	} else {
		w.runWithDynamicNumberOfWorker()
	}
}

func (w *worker) runWithDynamicNumberOfWorker() {

	keepAliveTime := w.workerPool.poolConf.KeepAliveTimeInMillis * time.Millisecond
	ticker := time.NewTicker(keepAliveTime)

	for {
		select {
		case <-w.workerPool.quitNow:
			ticker.Stop()
			logrus.Debugf("worker [%s] has stopped working.", w.id.String())
			w.workerPool.AddNumberOfCurrentAndIdleWorker(-1)
			return
		case job, isOpen := <-w.workerPool.jobQueue:
			ticker.Stop()

			if !isOpen {
				w.workerPool.AddNumberOfCurrentAndIdleWorker(-1)
				logrus.Debugf("worker[%s] has done its job.", w.id.String())
				return
			}

			w.doJob(job)

			ticker = time.NewTicker(keepAliveTime)
		case <-ticker.C:
			ticker.Stop()

			if w.workerPool.CompareAndDecrementCurrentWorker() {
				logrus.Debugf("worker [%s] has killed itself.", w.id.String())
				return
			}

			ticker = time.NewTicker(keepAliveTime)

		}
	}
}

func (w *worker) runWithFixedNumberOfWorker() {

	for {
		select {
		case <-w.workerPool.quitNow:
			logrus.Debugf("worker [%s] has stopped working.", w.id.String())
			w.workerPool.AddNumberOfCurrentAndIdleWorker(-1)
			return
		case job, isOpen := <-w.workerPool.jobQueue:
			if !isOpen {
				w.workerPool.AddNumberOfCurrentAndIdleWorker(-1)
				logrus.Debugf("worker[%s] has done its job.", w.id.String())
				return
			}

			w.doJob(job)
		}
	}
}
