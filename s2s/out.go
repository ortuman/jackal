/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"sync/atomic"

	"bytes"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
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
	domain  string
	state   uint32
	tr      transport.Transport
	parser  *xml.Parser
	actorCh chan func()
}

func NewOut(domain string, tr transport.Transport) *Out {
	s := &Out{
		domain:  domain,
		tr:      tr,
		parser:  xml.NewParser(tr, 32768),
		state:   connecting,
		actorCh: make(chan func(), streamMailboxSize),
	}
	go s.actorLoop()
	go s.doRead() // start reading transport...

	return s
}

func (s *Out) ID() string {
	return s.domain
}

func (s *Out) SendElement(elem xml.XElement) {
}

func (s *Out) Disconnect(err error) {
}

func (s *Out) StartSession() {
	s.actorCh <- func() {
		s.startSession()
	}
}

func (s *Out) startSession() {
	var ops *xml.Element
	var includeClosing bool

	buf := &bytes.Buffer{}
	ops = xml.NewElementName("stream:stream")
	ops.SetAttribute("xmlns", jabberServerNamespace)
	ops.SetAttribute("xmlns:stream", streamNamespace)
	buf.WriteString(`<?xml version="1.0"?>`)

	ops.SetAttribute("from", router.Instance().DefaultLocalDomain())
	ops.SetAttribute("to", s.ID())
	ops.SetAttribute("version", "1.0")
	ops.ToXML(buf, includeClosing)

	openStr := buf.String()
	log.Debugf("SEND: %s", openStr)

	s.tr.WriteString(buf.String())
}

func (s *Out) actorLoop() {
	for {
		f := <-s.actorCh
		f()
		if s.getState() == disconnected {
			return
		}
	}
}

func (s *Out) readElement(elem xml.XElement) {
	if elem != nil {
		log.Debugf("RECV: %v", elem)
		s.handleElement(elem)
	}
	if s.getState() != disconnected {
		go s.doRead()
	}
}

func (s *Out) handleElement(elem xml.XElement) {
}

func (s *Out) doRead() {
	if elem, err := s.parser.ParseElement(); err == nil {
		s.actorCh <- func() {
			s.readElement(elem)
		}
	} else {
		if s.getState() == disconnected {
			return // already disconnected...
		}
	}
}

func (s *Out) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *Out) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}
