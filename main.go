package pool

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrBadParam       = errors.New("bad params")
	ErrPoolNotStarted = errors.New("pool not started")
)

type TaskFunc func(any) (any, error)

type Task struct {
	F   TaskFunc
	Arg any
	Res any
	Err error
}

type WorkerPool struct {
	noWorkers   int
	started     bool
	start       sync.Once
	end         sync.Once
	workerGroup *sync.WaitGroup
	tasks       chan Task
	results     chan Task
}

func NewWorkerPool(noWorkers int) (*WorkerPool, error) {
	if noWorkers <= 0 {
		return nil, ErrBadParam
	}

	maxTasks := 1024
	return &WorkerPool{
		noWorkers:   noWorkers,
		started:     false,
		start:       sync.Once{},
		end:         sync.Once{},
		workerGroup: &sync.WaitGroup{},
		tasks:       make(chan Task, maxTasks),
		results:     make(chan Task, maxTasks),
	}, nil
}

func (wp *WorkerPool) Results() (<-chan Task, error) {
	if !wp.started {
		return nil, ErrPoolNotStarted
	}

	return wp.results, nil
}

func (wp *WorkerPool) Stop() {
	wp.end.Do(func() {
		close(wp.results)
	})
}

func (wp *WorkerPool) AddTasks(tasks []Task) {
	for _, t := range tasks {
		if t.F == nil {
			continue
		}

		wp.tasks <- t
	}
	close(wp.tasks)
}

func (wp *WorkerPool) Run(ctx context.Context) {
	wp.start.Do(func() {
		wp.started = true
		wp.startWorkers(ctx)
	})
}

func (wp *WorkerPool) worker(ctx context.Context) {
	defer wp.workerGroup.Done()

	for t := range wp.tasks {
		select {
		case <-ctx.Done():
			return
		default:
			t.Res, t.Err = t.F(t.Arg)
			wp.results <- t
		}
	}
}

func (wp *WorkerPool) startWorkers(ctx context.Context) {
	for i := 0; i < wp.noWorkers; i++ {
		wp.workerGroup.Add(1)
		go wp.worker(ctx)
	}

	go func() {
		wp.workerGroup.Wait()
		wp.Stop()
	}()
}
