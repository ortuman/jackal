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
	"strings"
	"sync/atomic"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

const (
	jabberClientNamespace = "jabber:client"
	jabberServerNamespace = "jabber:server"
	framedStreamNamespace = "urn:ietf:params:xml:ns:xmpp-framing"
	streamNamespace       = "http://etherx.jabber.org/streams"
)

// Error represents a session error.
type Error struct {
	// Element returns the original incoming element that generated
	// the session error.
	Element xml.XElement

	// UnderlyingErr is the underlying session error.
	UnderlyingErr error
}

// A Config structure is used to configure an XMPP session.
type Config struct {
	// JID defines an initial session JID.
	JID *xml.JID

	// Transport provides the underlying session transport
	// that will be used to send and received elements.
	Transport transport.Transport

	// MaxStanzaSize defines the maximum stanza size that
	// can be read from the session transport.
	MaxStanzaSize int

	// IsServer defines whether or not this session is established
	// between two servers.
	IsServer bool
}

// Session represents an XMPP session between the two peers.
type Session struct {
	id       string
	tr       transport.Transport
	pr       *xml.Parser
	isServer bool
	opened   uint32
	started  uint32
	sJID     atomic.Value
}

// New creates a new session instance.
func New(config *Config) *Session {
	var parsingMode xml.ParsingMode
	switch config.Transport.Type() {
	case transport.Socket:
		parsingMode = xml.SocketStream
	case transport.WebSocket:
		parsingMode = xml.WebSocketStream
	}
	s := &Session{
		id:       uuid.New(),
		tr:       config.Transport,
		pr:       xml.NewParser(config.Transport, parsingMode, config.MaxStanzaSize),
		isServer: config.IsServer,
	}
	s.sJID.Store(config.JID)
	return s
}

// UpdateJID updates current session JID.
func (s *Session) UpdateJID(sessionJID *xml.JID) {
	s.sJID.Store(sessionJID)
}

// Open opens session sending the proper XMPP payload.
func (s *Session) Open() error {
	if !atomic.CompareAndSwapUint32(&s.opened, 0, 1) {
		return errors.New("session already opened")
	}
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

	_, err := io.Copy(s.tr, strings.NewReader(openStr))
	return err
}

// Close closes session sending the proper XMPP payload.
// Is responsability of the caller to close underlying transport.
func (s *Session) Close() error {
	if atomic.LoadUint32(&s.opened) == 0 {
		return errors.New("session already closed")
	}
	switch s.tr.Type() {
	case transport.Socket:
		io.WriteString(s.tr, "</stream:stream>")
	case transport.WebSocket:
		io.WriteString(s.tr, fmt.Sprintf(`<close xmlns="%s" />`, framedStreamNamespace))
	}
	return nil
}

// Send writes an XML element to the underlying session transport.
func (s *Session) Send(elem xml.XElement) error {
	if atomic.LoadUint32(&s.opened) == 0 {
		return errors.New("send on closed session")
	}
	log.Debugf("SEND: %v", elem)
	elem.ToXML(s.tr, true)
	return nil
}

// Receive returns next incoming session element.
func (s *Session) Receive() (xml.XElement, *Error) {
	if atomic.LoadUint32(&s.opened) == 0 {
		return nil, &Error{UnderlyingErr: errors.New("receive from closed session")}
	}

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
	if len(to) > 0 && to != s.jid().Domain() {
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
	return s.sJID.Load().(*xml.JID)
}

func (s *Session) mapErrorToSessionError(err error) *Error {
	switch err {
	case nil, io.EOF, io.ErrUnexpectedEOF:
		break

	case xml.ErrStreamClosedByPeer:
		s.Close()

	case xml.ErrTooLargeStanza:
		return &Error{UnderlyingErr: streamerror.ErrPolicyViolation}

	default:
		switch err.(type) {
		case *stdxml.SyntaxError:
			return &Error{UnderlyingErr: streamerror.ErrInvalidXML}
		default:
			return &Error{UnderlyingErr: err}
		}
	}
	return &Error{}
}
