package core

import (
	//"fmt"
	"runtime/debug"
	"sync"
	"testing"
)

func AssertEqual[T comparable](test *testing.T, a T, b T) {
	if a != b {
		test.Errorf("\nFailure: %v != %v\n%s", a, b, string(debug.Stack()))
	}
}

func AssertPanics(test *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			test.Errorf("\nFailure: no panic occurred\n%s", string(debug.Stack()))
		}
	}()
	f()
}

func TestMakeSimpleQueue(t *testing.T) {
	sq := MakeSimpleQueue[int]()
	if sq.head != nil {
		t.Fail()
	}
	if sq.tail != nil {
		t.Fail()
	}
	if sq.mtx == nil {
		t.Fail()
	}
	if sm, ok := sq.mtx.(*sync.Mutex); ok {
		if sq.cv == nil {
			t.Fail()
		}
		if sc, ok := sq.cv.(*sync.Cond); ok {
			if sc.L == sm {
				return
			} else {
				t.Fail()
			}
		} else {
			t.Fail()
		}
	} else {
		t.Fail()
	}
}

type SqTestMutex struct {
	Test *testing.T
	LockCalls int
	UnlockCalls int
}

type SqTestCond struct {
	Mtx *SqTestMutex
	FailOnWait bool
	BroadcastCalls int
	WaitCalls int
}

func (m *SqTestMutex) Lock() {
	m.LockCalls++
}

func (m *SqTestMutex) Unlock() {
	m.UnlockCalls++
}

func (c *SqTestCond) Broadcast() {
	c.BroadcastCalls++
}

func (c *SqTestCond) Signal() {
	c.Mtx.Test.Fail()
}

func (c *SqTestCond) Wait() {
	c.WaitCalls++
	if c.FailOnWait {
		c.Mtx.Test.Log("\nSimpleQueue: Cond.Wait() was invoked, with FailOnWait = true\n" + string(debug.Stack()))
		c.Mtx.Test.FailNow()
	} else {
		panic("SimpleQueue: Cond.Wait() was invoked")
	}
}

func MakeTestMtxCond(t *testing.T, failOnWait bool) (*SqTestMutex, *SqTestCond) {
	m := &SqTestMutex{}
	m.Test = t
	c := &SqTestCond{}
	c.Mtx = m
	c.FailOnWait = failOnWait
	return m, c
}

func MakeTestSimpleQueue[T any](m *SqTestMutex, c *SqTestCond) SimpleQueue[T] {
	return SimpleQueue[T]{
		head: nil,
		tail: nil,
		mtx: m,
		cv: c,
	}
}

func TestSimpleQueueAddTake(t *testing.T) {
	m, c := MakeTestMtxCond(t, true)
	sq := MakeTestSimpleQueue[int](m, c)

	AssertEqual(t, c.BroadcastCalls, 0)
	AssertEqual(t, c.WaitCalls, 0)

	sq.Add(7)
	AssertEqual(t, c.BroadcastCalls, 1)
	sq.Add(5)
	sq.Add(6)
	AssertEqual(t, c.BroadcastCalls, 1)

	AssertEqual(t, sq.Take(), 7)
	AssertEqual(t, sq.Take(), 5)
	AssertEqual(t, sq.Take(), 6)
	AssertEqual(t, c.WaitCalls, 0)

	c.FailOnWait = false
	AssertPanics(t, func(){sq.Take()})
	AssertEqual(t, c.WaitCalls, 1)
}
