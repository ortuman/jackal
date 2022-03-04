// Copyright 2022 The jackal Authors
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

	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/pkg/router/stream"
)

type queueElement struct {
	st stravaganza.Stanza
	h  uint32
}

type queue struct {
	stm               stream.C2S
	nc                []byte
	reqAckInterval    time.Duration
	waitForAckTimeout time.Duration

	mu       sync.RWMutex
	elements []queueElement
	outH     uint32
	inH      uint32
	rTm      *time.Timer
	discTm   *time.Timer
}

func newQueue(
	stm stream.C2S,
	nonce []byte,
	requestAckInterval time.Duration,
	waitForAckTimeout time.Duration,
) *queue {
	sq := &queue{
		stm:               stm,
		nc:                nonce,
		reqAckInterval:    requestAckInterval,
		waitForAckTimeout: waitForAckTimeout,
	}
	sq.rTm = time.AfterFunc(requestAckInterval, sq.requestAck)
	return sq
}

func (q *queue) handleIn() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.setRTimer()
	q.inH = incH(q.inH)
}

func (q *queue) handleOut(stanza stravaganza.Stanza) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, el := range q.elements {
		if el.st == stanza {
			// stanza is being resent
			return
		}
	}
	q.outH = incH(q.outH)
	q.elements = append(q.elements, queueElement{
		st: stanza,
		h:  q.outH,
	})
}

func (q *queue) setStream(stm stream.C2S) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stm = stm
}

func (q *queue) stream() stream.C2S {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.stm
}

func (q *queue) nonce() []byte {
	return q.nc
}

func (q *queue) acknowledge(h uint32) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if discTm := q.discTm; discTm != nil {
		discTm.Stop() // cancel disconnection timeout
	}
	j := -1
	for i, e := range q.elements {
		if e.h <= h {
			j = i
		}
	}
	if j != -1 {
		q.elements = q.elements[j+1:]
	}
	q.setRTimer()
}

func (q *queue) sendPending() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	for _, e := range q.elements {
		q.stm.SendElement(e.st)
	}
}

func (q *queue) len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.elements)
}

func (q *queue) inboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.inH
}

func (q *queue) outboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.outH
}

func (q *queue) scheduleR() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.setRTimer()
}

func (q *queue) cancelTimers() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.rTm.Stop()
	if discTm := q.discTm; discTm != nil {
		discTm.Stop()
	}
}

func (q *queue) requestAck() {
	q.mu.Lock()
	defer q.mu.Unlock()

	r := stravaganza.NewBuilder("r").
		WithAttribute(stravaganza.Namespace, streamNamespace).
		Build()
	q.stm.SendElement(r)

	// schedule disconnect
	q.discTm = time.AfterFunc(q.waitForAckTimeout, func() {
		q.stm.Disconnect(streamerror.E(streamerror.ConnectionTimeout))
	})
}

func (q *queue) setRTimer() {
	q.rTm.Stop()
	q.rTm = time.AfterFunc(q.reqAckInterval, q.requestAck)
}

func incH(h uint32) uint32 {
	if h == math.MaxUint32-1 {
		return 0
	}
	return h + 1
}
