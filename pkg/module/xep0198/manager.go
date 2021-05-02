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

func incH(h uint32) uint32 {
	if h == math.MaxUint32-1 {
		return 0
	}
	return h + 1
}

type qEntry struct {
	el   stravaganza.Element
	outH uint32
}

type queue struct {
	entries []qEntry
	nextH   uint32
}

type manager struct {
	mu  sync.RWMutex
	q   queue
	inH uint32
}

func newManager() *manager {
	return &manager{
		q:   queue{nextH: 1},
		inH: 0,
	}
}

func (m *manager) pushElement(el stravaganza.Element) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.q.entries = append(m.q.entries, qEntry{
		el:   el,
		outH: m.q.nextH,
	})
	m.q.nextH = incH(m.q.nextH)
}

func (m *manager) acknowledge(h uint32) []stravaganza.Element {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i, e := range m.q.entries {
		if e.outH < h {
			continue
		}
		m.q.entries = m.q.entries[i+1:]
		break
	}
	var retVal []stravaganza.Element
	for _, qe := range m.q.entries {
		retVal = append(retVal, qe.el)
	}
	return retVal
}

func (m *manager) incInboundH() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inH++
}

func (m *manager) inboundH() uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inH
}
