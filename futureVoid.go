package fun

import (
	"runtime"
	"sync"
)

type FutureVoid struct {
	ch   chan struct{} // notification channel
	once sync.Once
}

// NewTask starts a new asynchronous task.
func NewFutureVoid(ctx Ctx, callback func()) *FutureVoid {
	ch := make(chan struct{}, 1)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				stackBuf := make([]byte, 8192)
				stackSize := runtime.Stack(stackBuf, false)
				stackTrace := string(stackBuf[:stackSize])
				fun.returnData(ctx.Id, ctx.RequestId, err, stackTrace)
			}
			close(ch)
		}()

		callback()
		ch <- struct{}{} // signal completion
	}()

	return &FutureVoid{ch: ch}
}

// Join blocks until the task completes.
func (t *FutureVoid) Join() {
	t.once.Do(func() {
		<-t.ch
	})
}

// AllTasks waits for all tasks to complete.
func AllFutureVoid(tasks ...*FutureVoid) {
	for _, task := range tasks {
		task.Join()
	}
}
