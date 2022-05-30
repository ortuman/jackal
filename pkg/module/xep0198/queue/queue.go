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

package streamqueue

import (
	"math"
	"sync"
	"time"

	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/pkg/router/stream"
)

const streamNamespace = "urn:xmpp:sm:3"

// QueueMap defines a map of stream stanza queues.
type QueueMap struct {
	mu     sync.RWMutex
	queues map[string]*Queue
}

// NewQueueMap creates and initializes a new QueueMap instance.
func NewQueueMap() *QueueMap {
	return &QueueMap{
		queues: make(map[string]*Queue),
	}
}

// Set associates a Queue value to k key.
func (qm *QueueMap) Set(k string, q *Queue) {
	qm.mu.Lock()
	qm.queues[k] = q
	qm.mu.Unlock()
}

// Get returns Queue associated to k key.
func (qm *QueueMap) Get(key string) *Queue {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return qm.queues[key]
}

// Delete deletes the Queue value associated to k key.
func (qm *QueueMap) Delete(k string) *Queue {
	qm.mu.Lock()
	q := qm.queues[k]
	if q != nil {
		delete(qm.queues, k)
	}
	qm.mu.Unlock()
	return q
}

// Element defines a stream queue element type.
type Element struct {
	// Stanza contains the element stanza.
	Stanza stravaganza.Stanza

	// H contains the incremental h value associated to the element stanza.
	H uint32
}

// Queue represents and c2s resumable queue.
type Queue struct {
	stm               stream.C2S
	nc                []byte
	reqAckInterval    time.Duration
	waitForAckTimeout time.Duration

	mu       sync.RWMutex
	elements []Element
	outH     uint32
	inH      uint32
	rTm      *time.Timer
	discTm   *time.Timer
}

// New creates and initializes a new Queue instance.
func New(
	stm stream.C2S,
	nonce []byte,
	elements []Element,
	inH uint32,
	outH uint32,
	requestAckInterval time.Duration,
	waitForAckTimeout time.Duration,
) *Queue {
	sq := &Queue{
		stm:               stm,
		nc:                nonce,
		elements:          elements,
		inH:               inH,
		outH:              outH,
		reqAckInterval:    requestAckInterval,
		waitForAckTimeout: waitForAckTimeout,
	}
	sq.rTm = time.AfterFunc(requestAckInterval, sq.RequestAck)
	return sq
}

// HandleIn process and incoming queue stanza.
func (q *Queue) HandleIn() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.setRTimer()
	q.inH = incH(q.inH)
}

// HandleOut process and outgoing queue stanza.
func (q *Queue) HandleOut(stanza stravaganza.Stanza) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, el := range q.elements {
		if el.Stanza == stanza {
			// stanza is being resent
			return
		}
	}
	q.outH = incH(q.outH)
	q.elements = append(q.elements, Element{
		Stanza: stanza,
		H:      q.outH,
	})
}

// SetStream sets queue internal stream.
func (q *Queue) SetStream(stm stream.C2S) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stm = stm
}

// GetStream returns queue internal stream.
func (q *Queue) GetStream() stream.C2S {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.stm
}

// Nonce returns the queue nonce byte slice.
func (q *Queue) Nonce() []byte {
	return q.nc
}

// Acknowledge process and acknowledge a h value.
func (q *Queue) Acknowledge(h uint32) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if discTm := q.discTm; discTm != nil {
		discTm.Stop() // cancel disconnection timeout
	}
	j := -1
	for i, e := range q.elements {
		if e.H <= h {
			j = i
		}
	}
	if j != -1 {
		q.elements = q.elements[j+1:]
	}
	q.setRTimer()
}

// SendPending sends all pending stanzas to the queue internal stream.
func (q *Queue) SendPending() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	for _, e := range q.elements {
		q.stm.SendElement(e.Stanza)
	}
}

// Elements returns queue pending to send elements.
func (q *Queue) Elements() []Element {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.elements
}

// Len returns current queue length.
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.elements)
}

// InboundH returns incoming h value.
func (q *Queue) InboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.inH
}

// OutboundH returns outgoing h value.
func (q *Queue) OutboundH() uint32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.outH
}

// ScheduleR schedules and r stanza sending.
func (q *Queue) ScheduleR() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.setRTimer()
}

// CancelTimers cancels all queue internal timers.
func (q *Queue) CancelTimers() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	q.rTm.Stop()
	if discTm := q.discTm; discTm != nil {
		discTm.Stop()
	}
}

// RequestAck sends an r stanza to the queue internal stream.
func (q *Queue) RequestAck() {
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

func (q *Queue) setRTimer() {
	q.rTm.Stop()
	q.rTm = time.AfterFunc(q.reqAckInterval, q.RequestAck)
}

func incH(h uint32) uint32 {
	if h == math.MaxUint32-1 {
		return 0
	}
	return h + 1
}
