/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package concurrent

import (
	"sync"
	"time"
)

type executorItem struct {
	f          func()
	continueCh chan struct{}
}

type Executor struct {
	sync.Mutex
	items chan executorItem
}

func (e *Executor) Sync(f func()) {
	item := executorItem{
		f:          f,
		continueCh: make(chan struct{}),
	}
	e.enqueueItem(item)
	<-item.continueCh
}

func (e *Executor) Async(f func()) {
	e.enqueueItem(executorItem{f: f})
}

func (e *Executor) enqueueItem(item executorItem) {
	e.Lock()
	if e.items == nil {
		e.items = make(chan executorItem, 256)
		go e.run()
	}
	e.items <- item
	e.Unlock()
}

func (e *Executor) run() {
	for {
		timeout := time.After(time.Second)
		select {
		case item := <-e.items:
			e.processItem(item)

		case <-timeout:
			e.Lock()
			// try reading after locking...
			select {
			case item := <-e.items:
				e.Unlock()
				e.processItem(item)
				continue
			default:
				close(e.items)
				e.items = nil
				e.Unlock()
				return
			}
		}
	}
}

func (e *Executor) processItem(item executorItem) {
	item.f()
	if item.continueCh != nil {
		close(item.continueCh)
	}
}
