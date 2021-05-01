// Copyright 2021 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0198

import (
	"sync"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type queueEntry struct {
	el stravaganza.Element
	h  uint64
}

type queue struct {
	mu      sync.RWMutex
	entries []queueEntry
	hNext   uint64
}

func newQueue() *queue {
	return &queue{hNext: 1}
}

func (q *queue) push(el stravaganza.Element) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, queueEntry{
		el: el,
		h:  q.hNext,
	})
	q.hNext++
}

func (q *queue) elements() []stravaganza.Element {
	q.mu.RLock()
	defer q.mu.RUnlock()
	var retVal []stravaganza.Element
	for _, e := range q.entries {
		retVal = append(retVal, e.el)
	}
	return retVal
}

func (q *queue) acknowledge(h uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, e := range q.entries {
		if e.h == h {
			continue
		}
		q.entries = append(q.entries[:i], q.entries[i+1:]...)
		break
	}
}
