/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package concurrent

import (
	"sync"
	"time"
)

type executorQueueItem struct {
	f          func()
	continueCh chan struct{}
}

type ExecutorQueue struct {
	sync.Mutex
	items chan executorQueueItem
}

func (eq *ExecutorQueue) Sync(f func()) {
	item := executorQueueItem{
		f:          f,
		continueCh: make(chan struct{}),
	}
	eq.enqueueItem(item)
	<-item.continueCh
}

func (eq *ExecutorQueue) Async(f func()) {
	eq.enqueueItem(executorQueueItem{f: f})
}

func (eq *ExecutorQueue) enqueueItem(item executorQueueItem) {
	eq.Lock()
	if eq.items == nil {
		eq.items = make(chan executorQueueItem, 256)
		go eq.run()
	}
	eq.items <- item
	eq.Unlock()
}

func (eq *ExecutorQueue) run() {
	for {
		timeout := time.After(time.Second)
		select {
		case item := <-eq.items:
			eq.processItem(item)

		case <-timeout:
			eq.Lock()
			// try reading after locking...
			select {
			case item := <-eq.items:
				eq.Unlock()
				eq.processItem(item)
				continue
			default:
				close(eq.items)
				eq.items = nil
				eq.Unlock()
				return
			}
		}
	}
}

func (eq *ExecutorQueue) processItem(item executorQueueItem) {
	item.f()
	if item.continueCh != nil {
		close(item.continueCh)
	}
}
