package pool

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestWorkerPool_NewPool(t *testing.T) {
	if _, err := NewWorkerPool(0); err != ErrBadParam {
		t.Fatalf("expected error when creating pool with 0 workers, got: %v", err)
	}

	if _, err := NewWorkerPool(-1); err != ErrBadParam {
		t.Fatalf("expected error when creating pool with -1 channel size, got: %v", err)
	}
}

func TestWorkerPool_MultipleStartStopDontPanic(t *testing.T) {
	wp, err := NewWorkerPool(5)
	if err != nil {
		t.Fatalf("error creating pool: %v", err)
	}

	// We're just checking to make sure multiple calls to start or stop
	// don't cause a panic
	ctx, cancel := context.WithTimeout(context.TODO(), 100*time.Millisecond)
	defer cancel()

	wp.Run(ctx)
	wp.Run(ctx)
	wp.Stop()
	wp.Stop()
}

func sillyFunc(xAny any) (any, error) {
	sum := int64(0)
	x := xAny.(int64)
	for i := int64(1); i <= x; i++ {
		sum += i
	}

	if sum%2 == 1 {
		return int64(0), errors.New("planned Execute() error")
	}

	return sum, nil
}

func TestWorkerPool_Work(t *testing.T) {
	var tasks []Task

	taskNumber := 200
	startFrom := int64(40000000)

	for i := range taskNumber {
		tasks = append(tasks, Task{F: sillyFunc, Arg: startFrom + int64(i)})
	}

	wp, err := NewWorkerPool(6)
	if err != nil {
		t.Fatalf("error making worker pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 1000*time.Millisecond)
	defer cancel()

	wp.Run(ctx)
	wp.AddTasks(tasks)

	resChannel, err := wp.Results()
	if err != nil {
		t.Fatalf("error fetching worker pool results: %v", err)
	}

	for t := range resChannel {
		arg := t.Arg.(int64)
		res := t.Res.(int64)
		errString := ""
		if t.Err != nil {
			errString = t.Err.Error()
		}

		fmt.Printf("%d -> (%d, %s)\n", arg, res, errString)
	}
}
