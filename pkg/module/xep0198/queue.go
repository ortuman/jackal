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
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const (
	requestAckInterval = time.Minute * 20
	waitForAckTimeout  = time.Second * 30
)

type stmQE struct {
	st stravaganza.Stanza
	h  uint32
}

type stmQ struct {
	stm   stream.C2S
	nonce []byte

	mu     sync.RWMutex
	q      []stmQE
	outH   uint32
	inH    uint32
	rTm    *time.Timer
	discTm *time.Timer
}

func newSQ(stm stream.C2S, nonce []byte) *stmQ {
	sq := &stmQ{
		stm:   stm,
		nonce: nonce,
	}
	sq.rTm = time.AfterFunc(requestAckInterval, sq.requestAck)
	return sq
}

func (q *stmQ) processInboundStanza() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.setRTimer()
	q.inH = incH(q.inH)
}

func (q *stmQ) processOutboundStanza(stanza stravaganza.Stanza) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.outH = incH(q.outH)
	q.q = append(q.q, stmQE{
		st: stanza,
		h:  q.outH,
	})
}

func (q *stmQ) acknowledge(h uint32) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if discTm := q.discTm; discTm != nil {
		discTm.Stop() // cancel disconnection timeout
	}
	for i, e := range q.q {
		if e.h < h {
			continue
		}
		q.q = q.q[i+1:]
		break
	}
	q.setRTimer()
}

func (q *stmQ) stanzas() []stravaganza.Stanza {
	q.mu.RLock()
	defer q.mu.RUnlock()
	var retVal []stravaganza.Stanza
	for _, e := range q.q {
		retVal = append(retVal, e.st)
	}
	return retVal
}

func (q *stmQ) inboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.inH
}

func (q *stmQ) outboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.outH
}

func (q *stmQ) scheduleRTimer() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.setRTimer()
}

func (q *stmQ) cancelRTimer() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.rTm.Stop()
}

func (q *stmQ) cancelTimers() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.rTm.Stop()

	if discTm := q.discTm; discTm != nil {
		discTm.Stop()
	}
}

func (q *stmQ) requestAck() {
	q.mu.Lock()
	defer q.mu.Unlock()

	r := stravaganza.NewBuilder("r").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		Build()
	q.stm.SendElement(r)

	// schedule disconnect
	q.discTm = time.AfterFunc(waitForAckTimeout, func() {
		q.stm.Disconnect(streamerror.E(streamerror.ConnectionTimeout))
	})
}

func (q *stmQ) setRTimer() {
	q.rTm.Stop()
	q.rTm = time.AfterFunc(requestAckInterval, q.requestAck)
}

func incH(h uint32) uint32 {
	if h == math.MaxUint32-1 {
		return 0
	}
	return h + 1
}
