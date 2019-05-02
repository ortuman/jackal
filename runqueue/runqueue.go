package runqueue

import (
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
}

func New(name string) *RunQueue {
	return &RunQueue{
		name:  name,
		queue: mpsc.New(),
	}
}

func (m *RunQueue) Post(fn func()) {
	m.queue.Push(fn)
	atomic.AddInt32(&m.messageCount, 1)
	m.schedule()
}

func (m *RunQueue) schedule() {
	if atomic.CompareAndSwapInt32(&m.state, idle, running) {
		go m.process()
	}
}

func (m *RunQueue) process() {

process:
	m.run()

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
			log.Debugf("run queue %s panicked with error: %v", m.name, err)
		}
	}()

	for {
		if fn := m.queue.Pop(); fn != nil {
			fn.(func())()
			atomic.AddInt32(&m.messageCount, -1)
		} else {
			return
		}
	}
}
