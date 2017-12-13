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

type OperationQueue struct {
	QueueSize int
	Timeout   time.Duration

	sync.Mutex
	items chan func()
}

func (oq *OperationQueue) Exec(f func()) {
	oq.enqueueItem(f)
}

func (oq *OperationQueue) enqueueItem(item func()) {
	oq.Lock()
	if oq.items == nil {
		var queueSize int
		if oq.QueueSize > 0 {
			queueSize = oq.QueueSize
		} else {
			queueSize = defaultQueueSize
		}
		oq.items = make(chan func(), queueSize)
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
		f := <-oq.items
		f()
	}
}

func (oq *OperationQueue) processItemsWithTimeout(timeout time.Duration) {
	for {
		timeout := time.After(time.Second)
		select {
		case f := <-oq.items:
			f()

		case <-timeout:
			oq.Lock()
			// try reading after locking...
			select {
			case f := <-oq.items:
				oq.Unlock()
				f()
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
