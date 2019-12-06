/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package runqueue

import (
	"runtime"
	"sync/atomic"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/runqueue/mpsc"
)

const (
	idle int32 = iota
	running
)

type RunQueue struct {
	name         string
	queue        *mpsc.Queue
	messageCount int32
	state        int32
	stopped      int32
}

type funcMessage struct{ fn func() }
type stopMessage struct{ stopCb func() }

func New(name string) *RunQueue {
	return &RunQueue{
		name:  name,
		queue: mpsc.New(),
	}
}

func (m *RunQueue) Run(fn func()) {
	if atomic.LoadInt32(&m.stopped) == 1 {
		return
	}
	m.queue.Push(&funcMessage{fn: fn})
	atomic.AddInt32(&m.messageCount, 1)
	m.schedule()
}

func (m *RunQueue) Stop(stopCb func()) {
	if atomic.CompareAndSwapInt32(&m.stopped, 0, 1) {
		if atomic.LoadInt32(&m.messageCount) > 0 {
			m.queue.Push(&stopMessage{stopCb: stopCb})
			return
		}
	}
	stopCb()
	return
}

func (m *RunQueue) schedule() {
	if atomic.CompareAndSwapInt32(&m.state, idle, running) {
		go m.process()
	}
}

func (m *RunQueue) process() {

process:
	m.run()

	if atomic.LoadInt32(&m.stopped) == 1 {
		return
	}

	atomic.StoreInt32(&m.state, idle)
	if atomic.LoadInt32(&m.messageCount) > 0 {
		// try setting the queue back to running
		if atomic.CompareAndSwapInt32(&m.state, idle, running) {
			goto process
		}
	}
}

func (m *RunQueue) run() {
	defer func() {
		if err := recover(); err != nil {
			stackSlice := make([]byte, 1024)
			s := runtime.Stack(stackSlice, false)
			log.Errorf("runqueue '%s' panicked with error: %v", m.name, err)
			log.Errorf("\n%s", stackSlice[0:s])
		}
	}()

	for {
		switch msg := m.queue.Pop().(type) {
		case *funcMessage:
			msg.fn()
			atomic.AddInt32(&m.messageCount, -1)
		case *stopMessage:
			if cb := msg.stopCb; cb != nil {
				cb()
			}
			return
		default:
			return
		}
	}
}
