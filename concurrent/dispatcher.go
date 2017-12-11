/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package concurrent

import (
	"sync"
	"time"
)

type dispatchQueueItem struct {
	f          func()
	continueCh chan struct{}
}

type DispatcherQueue struct {
	sync.Mutex
	items chan dispatchQueueItem
}

func (d *DispatcherQueue) Sync(f func()) {
	item := dispatchQueueItem{
		f:          f,
		continueCh: make(chan struct{}),
	}
	d.enqueueItem(item)
	<-item.continueCh
}

func (d *DispatcherQueue) Async(f func()) {
	d.enqueueItem(dispatchQueueItem{f: f})
}

func (d *DispatcherQueue) enqueueItem(item dispatchQueueItem) {
	d.Lock()
	if d.items == nil {
		d.items = make(chan dispatchQueueItem, 256)
		go d.run()
	}
	d.items <- item
	d.Unlock()
}

func (d *DispatcherQueue) run() {
	for {
		timeout := time.After(time.Second)
		select {
		case item := <-d.items:
			d.processItem(item)

		case <-timeout:
			d.Lock()
			// try reading after locking...
			select {
			case item := <-d.items:
				d.Unlock()
				d.processItem(item)
				continue
			default:
				close(d.items)
				d.items = nil
				d.Unlock()
				return
			}
		}
	}
}

func (d *DispatcherQueue) processItem(item dispatchQueueItem) {
	item.f()
	if item.continueCh != nil {
		close(item.continueCh)
	}
}
