package fun

import (
	"fmt"
	"runtime"
	"sync"
)

type FutureVoid struct {
	ch   chan struct{} // notification channel
	once sync.Once
	err  error // 添加错误字段来存储错误信息
}

// NewFutureVoid starts a new asynchronous task.
func NewFutureVoid(callback func()) *FutureVoid {
	ch := make(chan struct{}, 1)
	fv := &FutureVoid{ch: ch}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				stackBuf := make([]byte, 8192)
				stackSize := runtime.Stack(stackBuf, false)
				stackTrace := string(stackBuf[:stackSize])
				if value, ok := err.(Result[any]); ok {
					InfoLogger(err)
				} else {
					ErrorLogger(getErrorString(value) + "\n" + stackTrace)
				}
				if e, ok := err.(error); ok {
					fv.err = e
				} else {
					fv.err = fmt.Errorf("%v", err)
				}
				logWg.Wait()
			}
			close(ch)
		}()

		// 修复：调用回调函数并处理返回的错误
		callback()

		ch <- struct{}{} // signal completion
	}()

	return fv
}

// Join blocks until the task completes and returns any error that occurred.
func (t *FutureVoid) Join() error {
	t.once.Do(func() {
		<-t.ch
	})
	return t.err
}

func (t *FutureVoid) Then(callback func(error)) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				stackBuf := make([]byte, 8192)
				stackSize := runtime.Stack(stackBuf, false)
				stackTrace := string(stackBuf[:stackSize])
				if value, ok := err.(Result[any]); ok {
					InfoLogger(err)
				} else {
					ErrorLogger(getErrorString(value) + "\n" + stackTrace)
				}
				logWg.Wait()
			}
		}()
		err := t.Join()
		callback(err)
	}()
}

// AllFutureVoid waits for all tasks to complete and returns any error that occurred.
func AllFutureVoid(tasks ...*FutureVoid) []error {
	errors := make([]error, len(tasks))

	for i, task := range tasks {
		if err := task.Join(); err != nil {
			errors[i] = err
		}
	}

	return errors
}
