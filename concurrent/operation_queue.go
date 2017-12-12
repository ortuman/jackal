/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package concurrent

import (
	"sync"
	"time"
)

const defaultQueueSize = 256

type operationQueueItem struct {
	f          func()
	continueCh chan struct{}
}

type OperationQueue struct {
	QueueSize int
	Timeout   time.Duration

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
		var queueSize int
		if oq.QueueSize > 0 {
			queueSize = oq.QueueSize
		} else {
			queueSize = defaultQueueSize
		}
		oq.items = make(chan operationQueueItem, queueSize)
		go oq.run(oq.Timeout)
	}
	oq.items <- item
	oq.Unlock()
}

func (oq *OperationQueue) run(timeout time.Duration) {
	if timeout > 0 {
		oq.processItemsWithTimeout(timeout)
	} else {
		oq.processItems()
	}
}

func (oq *OperationQueue) processItems() {
	for {
		oq.processItem(<-oq.items)
	}
}

func (oq *OperationQueue) processItemsWithTimeout(timeout time.Duration) {
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
