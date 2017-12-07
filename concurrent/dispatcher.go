/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package concurrent

import (
	"sync"
)

type dispatchQueueItem struct {
	f          func()
	continueCh chan struct{}
}

type DispatcherQueue struct {
	sync.Mutex
	items  []dispatchQueueItem
	active bool
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
	d.items = append(d.items, item)
	if !d.active {
		d.active = true
		go d.run()
	}
	d.Unlock()
}

func (d *DispatcherQueue) run() {
	d.Lock()
	item := d.items[len(d.items)-1]
	d.items = d.items[:len(d.items)-1]
	d.Unlock()

	for {
		item.f()
		if item.continueCh != nil {
			close(item.continueCh)
		}

		d.Lock()
		if len(d.items) == 0 {
			d.active = false
			d.Unlock()
			return
		}
		item = d.items[len(d.items)-1]
		d.items = d.items[:len(d.items)-1]
		d.Unlock()
	}
}
