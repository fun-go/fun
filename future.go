package fun

import (
	"runtime"
	"sync"
)

type Future[T any] struct {
	ch    chan T
	value T
	once  sync.Once // 保证只执行一次
	ctx   Ctx
}

func NewFuture[T any](ctx Ctx, callback func() T) *Future[T] {
	ch := make(chan T, 1)
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

		value := callback()
		ch <- value
	}()

	return &Future[T]{
		ch:  ch,
		ctx: ctx,
	}
}

func (f *Future[T]) Join() T {
	f.once.Do(func() {
		f.value = <-f.ch
	})
	return f.value
}

func (f *Future[T]) Then(callback func(T)) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				stackBuf := make([]byte, 8192)
				stackSize := runtime.Stack(stackBuf, false)
				stackTrace := string(stackBuf[:stackSize])
				fun.returnData(f.ctx.Id, f.ctx.RequestId, err, stackTrace)
			}
		}()
		callback(f.Join())
	}()
}

// AllFuture 并行等待所有完成
// AllFuture 等待多个 Future 完成，直接返回结果切片（阻塞）
func AllFuture[T any](futures ...*Future[T]) []T {
	if len(futures) == 0 {
		panic(callError("fun: AllFuture: no futures provided"))
	}
	results := make([]T, len(futures))
	for i, f := range futures {
		results[i] = f.Join()
	}
	return results
}
