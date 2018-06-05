/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"sync/atomic"

	"bytes"

	"fmt"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xml"
)

const streamMailboxSize = 64

const (
	idle uint32 = iota
	connecting
	connected
	securing
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
		state:   idle,
		actorCh: make(chan func(), streamMailboxSize),
	}
	go s.actorLoop()
	go s.doRead() // start reading transport...

	return s
}

func (s *Out) Domain() string {
	return s.domain
}

func (s *Out) SendElement(elem xml.XElement) {
	s.actorCh <- func() {
		s.writeElement(elem)
	}
}

func (s *Out) Disconnect(err error) {
	waitCh := make(chan struct{})
	s.actorCh <- func() {
		s.disconnect(err)
		close(waitCh)
	}
	<-waitCh
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

	ops.SetAttribute("to", s.domain)
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

func (s *Out) handleElement(elem xml.XElement) {
	switch s.getState() {
	case connecting:
		s.handleConnecting(elem)
	case connected:
		s.handleConnected(elem)
	}
}

func (s *Out) handleConnecting(elem xml.XElement) {
	switch elem.Name() {
	case "stream:stream":
		if len(elem.Namespace()) > 0 && elem.Namespace() != jabberServerNamespace {
			s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
			return
		}
		if elem.Attributes().Get("version") != "1.0" {
			s.disconnectWithStreamError(streamerror.ErrUnsupportedVersion)
			return
		}
	}
	s.setState(connected)
}

func (s *Out) handleConnected(elem xml.XElement) {
	fmt.Println(elem)
}

func (s *Out) disconnect(err error) {
	if s.getState() == disconnected {
		return
	}
	switch err {
	case nil:
		s.disconnectClosingStream(false)
	default:
		if strmErr, ok := err.(*streamerror.Error); ok {
			s.disconnectWithStreamError(strmErr)
		} else {
			log.Error(err)
			s.disconnectClosingStream(false)
		}
	}
}

func (s *Out) writeElement(element xml.XElement) {
	log.Debugf("SEND: %v", element)
	s.tr.WriteElement(element, true)
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

func (s *Out) disconnectWithStreamError(err *streamerror.Error) {
	s.writeElement(err.Element())
	s.disconnectClosingStream(true)
}

func (s *Out) disconnectClosingStream(closeStream bool) {
	if closeStream {
		s.tr.WriteString("</stream:stream>")
	}
	// TODO(ortuman): unregister from router manager

	s.setState(disconnected)
	s.tr.Close()
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
