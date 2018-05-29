/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"sync/atomic"

	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xml"
)

const streamMailboxSize = 64

const (
	connecting uint32 = iota
	connected
	disconnected
)

type Out struct {
	id      string
	tr      transport.Transport
	state   uint32
	actorCh chan func()
}

func NewOut(identitifer string, tr transport.Transport) *Out {
	o := &Out{
		id:      identitifer,
		tr:      tr,
		state:   connecting,
		actorCh: make(chan func(), streamMailboxSize),
	}
	go o.actorLoop()
	return o
}

func (o *Out) ID() string {
	return o.id
}

func (o *Out) SendElement(elem xml.XElement) {
}

func (o *Out) Disconnect(err error) {
}

func (o *Out) StartSession() {
	o.actorCh <- func() {
		o.startSession()
	}
}

func (o *Out) startSession() {
}

func (o *Out) actorLoop() {
	for {
		f := <-o.actorCh
		f()
		if o.getState() == disconnected {
			return
		}
	}
}

func (o *Out) setState(state uint32) {
	atomic.StoreUint32(&o.state, state)
}

func (o *Out) getState() uint32 {
	return atomic.LoadUint32(&o.state)
}
