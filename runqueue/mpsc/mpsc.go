// This implementation is based on https://github.com/AsynkronIT/protoactor-go/blob/dev/internal/queue/mpsc/mpsc.go

// Package mpsc provides an efficient implementation of a multi-producer, single-consumer lock-free queue.
//
// The Push function is safe to call from multiple goroutines. The Pop and Empty APIs must only be
// called from a single, consumer goroutine.
//
package mpsc

import (
	"sync/atomic"
	"unsafe"
)

type node struct {
	next *node
	val  interface{}
}

type Queue struct {
	head *node
	tail *node
	stub node
}

func New() *Queue {
	q := &Queue{}
	q.head = &q.stub
	q.tail = &q.stub
	return q
}

// Push adds x to the back of the queue.
//
// Push can be safely called from multiple goroutines
func (q *Queue) Push(x interface{}) {
	n := &node{val: x}
	// current producer acquires head node
	prev := (*node)(atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.head)), unsafe.Pointer(n)))

	// release node to consumer
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&prev.next)), unsafe.Pointer(n))
}

// Pop removes the item from the front of the queue or nil if the queue is empty
//
// Pop must be called from a single, consumer goroutine
func (q *Queue) Pop() interface{} {
	tail := q.tail
	next := (*node)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&tail.next)))) // acquire
	if next != nil {
		q.tail = next
		v := next.val
		next.val = nil
		return v
	}
	return nil
}

// Empty returns true if the queue is empty
//
// Empty must be called from a single, consumer goroutine
func (q *Queue) Empty() bool {
	tail := q.tail
	next := (*node)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&tail.next))))
	return next == nil
}
