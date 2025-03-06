package core

import (
	"sync"
)

// The sync package doesn't have an interface for Cond, so we use our own
type SqCondition interface {
	Broadcast()
	Signal()
	Wait()
}

type QueueItem[T any] struct {
	value T
	next *QueueItem[T]
}

type SimpleQueue[T any] struct {
	head *QueueItem[T]
	tail *QueueItem[T]
	mtx sync.Locker
	cv SqCondition
}

func MakeSimpleQueue[T any]() *SimpleQueue[T] {
	m := &sync.Mutex{}
	return &SimpleQueue[T]{
		head: nil,
		tail: nil,
		mtx: m,
		cv: sync.NewCond(m),
	}
}

func (q *SimpleQueue[T]) Add(elem T) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	newItem := &QueueItem[T]{
		value: elem,
		next: nil,
	}

	if q.tail != nil {
		q.tail.next = newItem
	}
	q.tail = newItem

	if q.head == nil {
		q.head = newItem
		q.cv.Broadcast()
	}
}

func (q *SimpleQueue[T]) Take() T {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	for q.head == nil {
		q.cv.Wait()
	}

	item := q.head
	q.head = q.head.next
	return item.value
}
