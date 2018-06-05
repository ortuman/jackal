/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package session

import (
	"bytes"
	stdxml "encoding/xml"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const (
	jabberClientNamespace = "jabber:client"
	jabberServerNamespace = "jabber:server"
	framedStreamNamespace = "urn:ietf:params:xml:ns:xmpp-framing"
	streamNamespace       = "http://etherx.jabber.org/streams"
)

type Error struct {
	Element       xml.XElement
	UnderlyingErr error
}

type Config struct {
	JID       *xml.JID
	Transport transport.Transport
	Parser    *xml.Parser
	IsServer  bool
}

type Session struct {
	id       string
	tr       transport.Transport
	pr       *xml.Parser
	isServer bool
	started  uint32
	j        atomic.Value
}

func New(config *Config) *Session {
	s := &Session{
		id:       uuid.New(),
		tr:       config.Transport,
		pr:       config.Parser,
		isServer: config.IsServer,
	}
	s.j.Store(config.JID)
	return s
}

func (s *Session) UpdateJID(sessionJID *xml.JID) {
	s.j.Store(sessionJID)
}

func (s *Session) Open() error {
	var ops *xml.Element
	var includeClosing bool

	buf := &bytes.Buffer{}
	switch s.tr.Type() {
	case transport.Socket:
		ops = xml.NewElementName("stream:stream")
		ops.SetAttribute("xmlns", s.namespace())
		ops.SetAttribute("xmlns:stream", streamNamespace)
		buf.WriteString(`<?xml version="1.0"?>`)

	case transport.WebSocket:
		ops = xml.NewElementName("open")
		ops.SetAttribute("xmlns", framedStreamNamespace)
		includeClosing = true

	default:
		return nil
	}
	ops.SetAttribute("id", s.id)
	ops.SetAttribute("from", s.jid().Domain())
	ops.SetAttribute("version", "1.0")
	ops.ToXML(buf, includeClosing)

	openStr := buf.String()
	log.Debugf("SEND: %s", openStr)

	if err := s.tr.WriteString(buf.String()); err != nil {
		return err
	}
	return nil
}

func (s *Session) Close() error {
	switch s.tr.Type() {
	case transport.Socket:
		s.tr.WriteString("</stream:stream>")
	case transport.WebSocket:
		s.tr.WriteString(fmt.Sprintf(`<close xmlns="%s"/>`, framedStreamNamespace))
	}
	return nil
}

func (s *Session) Send(elem xml.XElement) {
	log.Debugf("SEND: %v", elem)
	s.tr.WriteElement(elem, true)
}

func (s *Session) Receive() (xml.XElement, *Error) {
	elem, err := s.pr.ParseElement()
	if err != nil {
		return nil, s.mapErrorToSessionError(err)
	} else if elem != nil {
		log.Debugf("RECV: %v", elem)

		if atomic.LoadUint32(&s.started) == 0 {
			if err := s.validateStreamElement(elem); err != nil {
				return nil, err
			}
			atomic.StoreUint32(&s.started, 1)

		} else if elem.IsStanza() {
			stanza, err := s.buildStanza(elem)
			if err != nil {
				return nil, err
			}
			return stanza, nil

		} else {
			isWebSocketTr := s.tr.Type() == transport.WebSocket
			if isWebSocketTr && elem.Name() == "close" && elem.Namespace() == framedStreamNamespace {
				return nil, &Error{}
			}
		}
	}
	return elem, nil
}

func (s *Session) buildStanza(elem xml.XElement) (xml.Stanza, *Error) {
	if err := s.validateNamespace(elem); err != nil {
		return nil, err
	}
	fromJID, toJID, err := s.extractAddresses(elem)
	if err != nil {
		return nil, err
	}
	switch elem.Name() {
	case "iq":
		iq, err := xml.NewIQFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, &Error{Element: elem, UnderlyingErr: xml.ErrBadRequest}
		}
		return iq, nil

	case "presence":
		presence, err := xml.NewPresenceFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, &Error{Element: elem, UnderlyingErr: xml.ErrBadRequest}
		}
		return presence, nil

	case "message":
		message, err := xml.NewMessageFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, &Error{Element: elem, UnderlyingErr: xml.ErrBadRequest}
		}
		return message, nil
	}
	return nil, &Error{UnderlyingErr: streamerror.ErrUnsupportedStanzaType}
}

func (s *Session) extractAddresses(elem xml.XElement) (*xml.JID, *xml.JID, *Error) {
	var fromJID, toJID *xml.JID
	var err error

	// do not validate 'from' address until full user JID has been set
	if s.jid().IsFullWithUser() {
		from := elem.From()
		if len(from) > 0 && !s.isValidFrom(from) {
			return nil, nil, &Error{UnderlyingErr: streamerror.ErrInvalidFrom}
		}
	}
	fromJID = s.jid()

	// validate 'to' address
	to := elem.To()
	if len(to) > 0 {
		toJID, err = xml.NewJIDString(elem.To(), false)
		if err != nil {
			return nil, nil, &Error{Element: elem, UnderlyingErr: xml.ErrJidMalformed}
		}
	} else {
		toJID = s.jid().ToBareJID() // account's bare JID as default 'to'
	}
	return fromJID, toJID, nil
}

func (s *Session) isValidFrom(from string) bool {
	validFrom := false
	j, err := xml.NewJIDString(from, false)
	if err == nil && j != nil {
		node := j.Node()
		domain := j.Domain()
		resource := j.Resource()

		validFrom = node == s.jid().Node() && domain == s.jid().Domain()
		if len(resource) > 0 {
			validFrom = validFrom && resource == s.jid().Resource()
		}
	}
	return validFrom
}

func (s *Session) validateStreamElement(elem xml.XElement) *Error {
	switch s.tr.Type() {
	case transport.Socket:
		if elem.Name() != "stream:stream" {
			return &Error{UnderlyingErr: streamerror.ErrUnsupportedStanzaType}
		}
		if elem.Namespace() != s.namespace() || elem.Attributes().Get("xmlns:stream") != streamNamespace {
			return &Error{UnderlyingErr: streamerror.ErrInvalidNamespace}
		}

	case transport.WebSocket:
		if elem.Name() != "open" {
			return &Error{UnderlyingErr: streamerror.ErrUnsupportedStanzaType}
		}
		if elem.Namespace() != framedStreamNamespace {
			return &Error{UnderlyingErr: streamerror.ErrInvalidNamespace}
		}
	}
	to := elem.To()
	if len(to) > 0 && !router.Instance().IsLocalDomain(to) {
		return &Error{UnderlyingErr: streamerror.ErrHostUnknown}
	}
	if elem.Version() != "1.0" {
		return &Error{UnderlyingErr: streamerror.ErrUnsupportedVersion}
	}
	return nil
}

func (s *Session) validateNamespace(elem xml.XElement) *Error {
	ns := elem.Namespace()
	if len(ns) == 0 || ns == s.namespace() {
		return nil
	}
	return &Error{UnderlyingErr: streamerror.ErrInvalidNamespace}
}

func (s *Session) namespace() string {
	if s.isServer {
		return jabberServerNamespace
	}
	return jabberClientNamespace
}

func (s *Session) jid() *xml.JID {
	return s.j.Load().(*xml.JID)
}

func (s *Session) mapErrorToSessionError(err error) *Error {
	switch err {
	case nil, io.EOF, io.ErrUnexpectedEOF:
		break

	case xml.ErrTooLargeStanza:
		return &Error{UnderlyingErr: streamerror.ErrPolicyViolation}

	case xml.ErrStreamClosedByPeer: // ...received </stream:stream>
		if s.tr.Type() != transport.Socket {
			return &Error{UnderlyingErr: streamerror.ErrInvalidXML}
		}

	default:
		switch err.(type) {
		case *stdxml.SyntaxError:
			return &Error{UnderlyingErr: streamerror.ErrInvalidXML}
		default:
			return &Error{UnderlyingErr: streamerror.ErrUndefinedCondition}
		}
	}
	return &Error{}
}
