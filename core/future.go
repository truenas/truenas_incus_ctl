package core

import (
	"sync"
	"time"
)

type Future[T any] struct {
	mtx      *sync.Mutex
	cv       *sync.Cond
	done_    bool
	wokenUp_ bool
	value_   T
	err_     error
}

func MakeFuture[T any]() *Future[T] {
	m := &sync.Mutex{}
	return &Future[T]{
		mtx: m,
		cv: sync.NewCond(m),
		done_: false,
		err_: nil,
	}
}

func MakeFutureArray[T any](count int) []*Future[T] {
	m := &sync.Mutex{}
	cv := sync.NewCond(m)
	arr := make([]*Future[T], count)
	for i := 0; i < count; i++ {
		arr[i] = &Future[T]{
			mtx: m,
			cv: cv,
			done_: false,
			err_: nil,
		}
	}
	return arr
}

func (f *Future[T]) Complete(value T) {
	f.Reach(value, nil)
}

func (f *Future[T]) Fail(err error) {
	var defaultValue T
	f.Reach(defaultValue, err)
}

func (f *Future[T]) Reach(value T, err error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	if !f.done_ {
		f.value_ = value
		f.err_ = err
		f.wokenUp_ = false
		f.done_ = true
		f.cv.Broadcast()
	}
}

func (f *Future[T]) Peek() (bool, T, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	if f.done_ {
		return true, f.value_, f.err_
	}

	var defaultValue T
	return false, defaultValue, nil
}

func (f *Future[T]) Get() (T, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	for !f.done_ {
		f.cv.Wait()
	}

	return f.value_, f.err_
}

func (f *Future[T]) MaybeGet() (bool, T, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	f.wokenUp_ = false
	for !f.done_ {
		f.cv.Wait()
		if f.wokenUp_ {
			f.wokenUp_ = false
			var defaultValue T
			return false, defaultValue, nil
		}
	}

	return true, f.value_, f.err_
}

func (f *Future[T]) Interrupt() {
	f.mtx.Lock()
	f.wokenUp_ = true
	f.cv.Broadcast()
	f.mtx.Unlock()
}

// sync.Cond.WaitTimeout(...) doesn't exist yet, so we burn a goroutine and hope for the best
func AwaitFutureOrTimeout[T any](f *Future[T], timeout time.Duration) (bool, T, error) {
	doneCh := make(chan []interface{})
	go func() {
		completed, value, err := f.MaybeGet()
		// Only send on the channel if we didn't time out, since if we did there'd be no one receiving so we'd block forever.
		if completed {
			doneCh <- []interface{}{value, err}
		}
		close(doneCh)
	}()

	select {
	case result := <-doneCh:
		var value T
		var err error
		if result[0] != nil {
			value = result[0].(T)
		}
		if result[1] != nil {
			err = result[1].(error)
		}
		return true, value, err
	case <-time.After(timeout):
		break
	}

	f.Interrupt()

	var defaultValue T
	return false, defaultValue, nil
}
