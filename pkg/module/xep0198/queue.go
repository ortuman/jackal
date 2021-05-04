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
	"math"
	"sync"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type qEntry struct {
	st stravaganza.Stanza
	h  uint32
}

type stanzaQueue struct {
	mu    sync.RWMutex
	queue []qEntry
	outH  uint32
	inH   uint32
}

func (q *stanzaQueue) push(st stravaganza.Stanza) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.outH = incH(q.outH)
	q.queue = append(q.queue, qEntry{
		st: st,
		h:  q.outH,
	})
}

func (q *stanzaQueue) acknowledge(h uint32) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, e := range q.queue {
		if e.h < h {
			continue
		}
		q.queue = q.queue[i+1:]
		break
	}
}

func (q *stanzaQueue) entries() []stravaganza.Stanza {
	q.mu.RLock()
	defer q.mu.RUnlock()
	var retVal []stravaganza.Stanza
	for _, qe := range q.queue {
		retVal = append(retVal, qe.st)
	}
	return retVal
}

func (q *stanzaQueue) len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue)
}

func (q *stanzaQueue) incInboundH() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.inH++
}

func (q *stanzaQueue) inboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.inH
}

func (q *stanzaQueue) outboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.outH
}

func incH(h uint32) uint32 {
	if h == math.MaxUint32-1 {
		return 0
	}
	return h + 1
}
