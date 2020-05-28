package worker_pool

type Job interface {
	Id() string
	Execute() error
}
