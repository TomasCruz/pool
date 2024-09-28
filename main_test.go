package pool

import (
	"context"
	"errors"
	"testing"
)

func TestWorkerPool_NewPool(t *testing.T) {
	if _, err := NewWorkerPool(0, 2); err != ErrBadParam {
		t.Fatalf("expected error when creating pool with 0 workers, got: %v", err)
	}

	if _, err := NewWorkerPool(11, 2); err != ErrBadParam {
		t.Fatalf("expected error when creating pool with 11 workers, got: %v", err)
	}

	if _, err := NewWorkerPool(-1, 2); err != ErrBadParam {
		t.Fatalf("expected error when creating pool with -1 channel size, got: %v", err)
	}
}

func TestWorkerPool_MultipleStartStopDontPanic(t *testing.T) {
	wp, err := NewWorkerPool(5, 5)
	if err != nil {
		t.Fatalf("error creating pool: %v", err)
	}

	// We're just checking to make sure multiple calls to start or stop
	// don't cause a panic
	wp.Run(context.TODO())
	wp.Run(context.TODO())
}

func sillyFunc(xInterface interface{}) (interface{}, error) {
	sum := int64(0)
	x := xInterface.(int64)
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

	taskNumber := 12
	startFrom := int64(400000000)

	for i := 0; i < taskNumber; i++ {
		tasks = append(tasks, Task{F: sillyFunc, Arg: startFrom + int64(i)})
	}

	wp, err := NewWorkerPool(4, taskNumber)
	if err != nil {
		t.Fatalf("error making worker pool: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Run(ctx)

	for _, t := range tasks {
		wp.AddTask(t)
	}

	resChannel, err := wp.Results()
	if err != nil {
		t.Fatalf("error fetching worker pool results: %v", err)
	}

	if len(resChannel) != taskNumber {
		t.Fatalf("resChannel length wrong: %v", len(resChannel))
	}

	// for t := range resChannel {
	// 	arg := t.Arg.(int64)
	// 	res := t.Res.(int64)
	// 	errString := ""
	// 	if t.Err != nil {
	// 		errString = t.Err.Error()
	// 	}

	// 	fmt.Printf("%d -> (%d, %s)\n", arg, res, errString)
	// }
}
