/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package concurrent

import (
	"sync"
	"time"
)

type operationQueueItem struct {
	f          func()
	continueCh chan struct{}
}

type OperationQueue struct {
	sync.Mutex
	items chan operationQueueItem
}

func (oq *OperationQueue) Sync(f func()) {
	item := operationQueueItem{
		f:          f,
		continueCh: make(chan struct{}),
	}
	oq.enqueueItem(item)
	<-item.continueCh
}

func (oq *OperationQueue) Async(f func()) {
	oq.enqueueItem(operationQueueItem{f: f})
}

func (oq *OperationQueue) enqueueItem(item operationQueueItem) {
	oq.Lock()
	if oq.items == nil {
		oq.items = make(chan operationQueueItem, 256)
		go oq.run()
	}
	oq.items <- item
	oq.Unlock()
}

func (oq *OperationQueue) run() {
	for {
		timeout := time.After(time.Second)
		select {
		case item := <-oq.items:
			oq.processItem(item)

		case <-timeout:
			oq.Lock()
			// try reading after locking...
			select {
			case item := <-oq.items:
				oq.Unlock()
				oq.processItem(item)
				continue
			default:
				close(oq.items)
				oq.items = nil
				oq.Unlock()
				return
			}
		}
	}
}

func (oq *OperationQueue) processItem(item operationQueueItem) {
	item.f()
	if item.continueCh != nil {
		close(item.continueCh)
	}
}
