package fun

import (
	"runtime"
	"sync"
)

type Future[T any] struct {
	ch    chan T
	value T
	once  sync.Once
	err   any // 添加错误字段来存储错误信息
}

func NewFuture[T any](callback func() T) *Future[T] {
	ch := make(chan T, 1)
	fv := &Future[T]{
		ch: ch,
	}

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
				fv.err = err
			}
			close(ch)
		}()

		// 调用回调函数并处理返回的错误
		value := callback()
		ch <- value
	}()

	return fv
}

func (f *Future[T]) Join() (T, any) {
	f.once.Do(func() {
		f.value = <-f.ch
	})
	return f.value, f.err
}

func (f *Future[T]) Then(callback func(T, any)) {
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
			}
		}()
		value, err := f.Join()
		callback(value, err)
	}()
}

// AllFuture 等待多个 Future 完成，返回结果切片和错误切片
type FutureAllType[T any] struct {
	Results []T
	Errors  []any
}

func AllFuture[T any](futures ...*Future[T]) FutureAllType[T] {
	results := make([]T, len(futures))
	var errors []any

	for i, f := range futures {
		value, err := f.Join()
		if err != nil {
			errors = append(errors, err)
		}
		results[i] = value
	}

	return FutureAllType[T]{
		Results: results,
		Errors:  errors,
	}
}
