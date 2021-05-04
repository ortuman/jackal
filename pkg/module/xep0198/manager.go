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
	"time"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const (
	requestAckInterval = time.Minute

	requestAckStanzaCount = 5
)

type qEntry struct {
	st stravaganza.Stanza
	h  uint32
}

type manager struct {
	stm stream.C2S

	mu    sync.RWMutex
	queue []qEntry
	outH  uint32
	inH   uint32
	tm    *time.Timer
}

func newManager(stm stream.C2S) *manager {
	m := &manager{stm: stm}
	m.tm = time.AfterFunc(requestAckInterval, m.requestAck)
	return m
}

func (m *manager) processInboundStanza() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tm.Stop()
	m.tm = time.AfterFunc(requestAckInterval, m.requestAck)
	m.inH = incH(m.inH)
}

func (m *manager) processOutboundStanza(stanza stravaganza.Stanza) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outH = incH(m.outH)
	m.queue = append(m.queue, qEntry{
		st: stanza,
		h:  m.outH,
	})
	if m.outH%requestAckStanzaCount == 0 {
		m.requestAck()
	}
}

func (m *manager) acknowledge(h uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.queue {
		if e.h < h {
			continue
		}
		m.queue = m.queue[i+1:]
		break
	}
}

func (m *manager) inboundH() uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inH
}

func (m *manager) cancelScheduledR() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tm.Stop()
}

func (m *manager) requestAck() {
	r := stravaganza.NewBuilder("r").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		Build()
	m.stm.SendElement(r)
	m.tm = time.AfterFunc(requestAckInterval, m.requestAck)
}

func incH(h uint32) uint32 {
	if h == math.MaxUint32-1 {
		return 0
	}
	return h + 1
}
