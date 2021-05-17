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
	"crypto/rand"
	"math"
	"sync"
	"time"

	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const (
	nonceLength = 16

	requestAckInterval = time.Minute * 2
	waitForAckTimeout  = time.Second * 30
)

type qEntry struct {
	st stravaganza.Stanza
	h  uint32
}

type queue struct {
	stm   stream.C2S
	nonce []byte

	mu     sync.RWMutex
	q      []qEntry
	outH   uint32
	inH    uint32
	tm     *time.Timer
	discTm *time.Timer
}

func newQueue(stm stream.C2S) (*queue, error) {
	m := &queue{
		stm:   stm,
		nonce: make([]byte, nonceLength),
	}
	// generate nonce
	_, err := rand.Read(m.nonce)
	if err != nil {
		return nil, err
	}
	m.tm = time.AfterFunc(requestAckInterval, m.requestAck)
	return m, nil
}

func (m *queue) processInboundStanza() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scheduleR()
	m.inH = incH(m.inH)
}

func (m *queue) processOutboundStanza(stanza stravaganza.Stanza) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outH = incH(m.outH)
	m.q = append(m.q, qEntry{
		st: stanza,
		h:  m.outH,
	})
}

func (m *queue) acknowledge(h uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if discTm := m.discTm; discTm != nil {
		discTm.Stop() // cancel disconnection timeout
	}
	for i, e := range m.q {
		if e.h < h {
			continue
		}
		m.q = m.q[i+1:]
		break
	}
	m.scheduleR()
}

func (m *queue) queue() []stravaganza.Stanza {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var retVal []stravaganza.Stanza
	for _, e := range m.q {
		retVal = append(retVal, e.st)
	}
	return retVal
}

func (m *queue) inboundH() uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inH
}

func (m *queue) outboundH() uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.outH
}

func (m *queue) cancelTimers() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tm.Stop()
	if discTm := m.discTm; discTm != nil {
		discTm.Stop()
	}
}

func (m *queue) requestAck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	r := stravaganza.NewBuilder("r").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		Build()
	m.stm.SendElement(r)

	// schedule disconnect
	m.discTm = time.AfterFunc(waitForAckTimeout, func() {
		m.stm.Disconnect(streamerror.E(streamerror.ConnectionTimeout))
	})
}

func (m *queue) scheduleR() {
	m.tm.Stop()
	m.tm = time.AfterFunc(requestAckInterval, m.requestAck)
}

func incH(h uint32) uint32 {
	if h == math.MaxUint32-1 {
		return 0
	}
	return h + 1
}
