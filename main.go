package pool

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrBadParam       = errors.New("bad params")
	ErrBadTask        = errors.New("bad task")
	ErrPoolNotStarted = errors.New("pool not started")
)

type TaskFunc func(interface{}) (interface{}, error)

type Task struct {
	F   TaskFunc
	Arg interface{}
	Res interface{}
	Err error
}

type WorkerPool struct {
	noWorkers   int
	maxTasks    int
	started     bool
	start       sync.Once
	end         sync.Once
	workersDone chan struct{}
	tasks       chan Task
	results     chan Task
}

func NewWorkerPool(noWorkers, maxTasks int) (*WorkerPool, error) {
	if noWorkers <= 0 || noWorkers > 10 {
		return nil, ErrBadParam
	}

	if maxTasks <= 0 || maxTasks > 100 {
		return nil, ErrBadParam
	}

	return &WorkerPool{
		noWorkers:   noWorkers,
		maxTasks:    maxTasks,
		started:     false,
		start:       sync.Once{},
		end:         sync.Once{},
		workersDone: make(chan struct{}),
		tasks:       make(chan Task, maxTasks),
		results:     make(chan Task, maxTasks),
	}, nil
}

func (wp *WorkerPool) Results() (<-chan Task, error) {
	if !wp.started {
		return nil, ErrPoolNotStarted
	}

	wp.end.Do(func() {
		close(wp.tasks)
		for i := 0; i < wp.noWorkers; i++ {
			<-wp.workersDone
		}
		close(wp.results)
	})

	return wp.results, nil
}

func (wp *WorkerPool) AddTask(task Task) error {
	if task.F == nil {
		return ErrBadTask
	}

	wp.tasks <- task
	return nil
}

func (wp *WorkerPool) Run(ctx context.Context) {
	wp.start.Do(func() {
		wp.started = true
		wp.startWorkers(ctx)
	})
}

func (wp *WorkerPool) worker(ctx context.Context) {
	for t := range wp.tasks {
		select {
		case <-ctx.Done():
			wp.workersDone <- struct{}{}
			return
		default:
			t.Res, t.Err = t.F(t.Arg)
			wp.results <- t
		}
	}

	wp.workersDone <- struct{}{}
}

func (wp *WorkerPool) startWorkers(ctx context.Context) {
	for i := 0; i < wp.noWorkers; i++ {
		go wp.worker(ctx)
	}
}
