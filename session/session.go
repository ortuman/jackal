/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package session

import (
	"context"
	stdxml "encoding/xml"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router/host"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

const (
	jabberClientNamespace = "jabber:client"
	jabberServerNamespace = "jabber:server"
	framedStreamNamespace = "urn:ietf:params:xml:ns:xmpp-framing"
	streamNamespace       = "http://etherx.jabber.org/streams"
	dialbackNamespace     = "jabber:server:dialback"
)

type namespaceSettable interface {
	SetNamespace(string)
}

// Error represents a session error.
type Error struct {
	// Element returns the original incoming element that generated
	// the session error.
	Element xmpp.XElement

	// UnderlyingErr is the underlying session error.
	UnderlyingErr error
}

// A Config structure is used to configure an XMPP session.
type Config struct {
	// JID defines an initial session JID.
	JID *jid.JID

	// MaxStanzaSize defines the maximum stanza size that
	// can be read from the session transport.
	MaxStanzaSize int

	// Remote domain represents the remote receiving entity domain name.
	RemoteDomain string

	// IsServer defines whether or not this session is established
	// by the server.
	IsServer bool

	// IsInitiating defines whether or not this is an initiating
	// entity session.
	IsInitiating bool
}

// Session represents an XMPP session between the two peers.
type Session struct {
	id           string
	hosts        *host.Hosts
	tr           transport.Transport
	pr           *xmpp.Parser
	remoteDomain string
	isServer     bool
	isInitiating bool
	opened       uint32
	started      uint32

	mu       sync.RWMutex
	streamID string
	sJID     *jid.JID
}

// New creates a new session instance.
func New(id string, config *Config, tr transport.Transport, hosts *host.Hosts) *Session {
	var parsingMode xmpp.ParsingMode
	switch tr.Type() {
	case transport.Socket:
		parsingMode = xmpp.SocketStream
	}
	s := &Session{
		id:           id,
		hosts:        hosts,
		tr:           tr,
		pr:           xmpp.NewParser(tr, parsingMode, config.MaxStanzaSize),
		remoteDomain: config.RemoteDomain,
		isServer:     config.IsServer,
		isInitiating: config.IsInitiating,
		sJID:         config.JID,
	}
	if !s.isInitiating {
		s.streamID = uuid.New()
	}
	return s
}

// StreamID returns session stream identifier.
func (s *Session) StreamID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamID
}

// SetJID updates current session JID.
func (s *Session) SetJID(sessionJID *jid.JID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sJID = sessionJID
}

// SetRemoteDomain sets current session remote domain.
func (s *Session) SetRemoteDomain(remoteDomain string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteDomain = remoteDomain
}

// Open initializes a sending the proper XMPP payload.
func (s *Session) Open(ctx context.Context, featuresElem xmpp.XElement) error {
	if !atomic.CompareAndSwapUint32(&s.opened, 0, 1) {
		return errors.New("session already opened")
	}
	var ops *xmpp.Element
	var includeClosing bool

	buf := &strings.Builder{}
	switch s.tr.Type() {
	case transport.Socket:
		ops = xmpp.NewElementName("stream:stream")
		ops.SetAttribute("xmlns", s.namespace())
		ops.SetAttribute("xmlns:stream", streamNamespace)
		if s.isServer {
			ops.SetAttribute("xmlns:db", dialbackNamespace)
		}
		buf.WriteString(`<?xml version="1.0"?>`)

	default:
		return nil
	}
	if !s.isInitiating {
		s.mu.RLock()
		ops.SetAttribute("id", s.streamID)
		s.mu.RUnlock()
	}
	ops.SetAttribute("from", s.jid().Domain())
	if s.isInitiating {
		s.mu.RLock()
		ops.SetAttribute("to", s.remoteDomain)
		s.mu.RUnlock()
	}
	ops.SetAttribute("version", "1.0")
	if err := ops.ToXML(buf, includeClosing); err != nil {
		return err
	}

	if featuresElem != nil {
		if err := featuresElem.ToXML(buf, true); err != nil {
			return err
		}
	}
	openStr := buf.String()
	log.Debugf("SEND(%s): %s", s.id, openStr)

	s.setWriteDeadline(ctx)

	_, err := io.Copy(s.tr, strings.NewReader(openStr))
	if err != nil {
		return err
	}
	return s.tr.Flush()
}

// Close closes session sending the proper XMPP payload.
// Is responsibility of the caller to close underlying transport.
func (s *Session) Close(ctx context.Context) error {
	if atomic.LoadUint32(&s.opened) == 0 {
		return errors.New("session already closed")
	}
	s.setWriteDeadline(ctx)

	var err error
	switch s.tr.Type() {
	case transport.Socket:
		_, err = io.WriteString(s.tr, "</stream:stream>")
	}
	if err != nil {
		return err
	}
	return s.tr.Flush()
}

// Send writes an XML element to the underlying session transport.
func (s *Session) Send(ctx context.Context, elem xmpp.XElement) error {
	// clear namespace if sending a stanza
	if e, ok := elem.(namespaceSettable); elem.IsStanza() && ok {
		e.SetNamespace("")
	}
	log.Debugf("SEND(%s): %v", s.id, elem)

	s.setWriteDeadline(ctx)

	if err := elem.ToXML(s.tr, true); err != nil {
		return err
	}
	return s.tr.Flush()
}

// Receive returns next incoming session element.
func (s *Session) Receive() (xmpp.XElement, *Error) {
	elem, err := s.pr.ParseElement()
	if err != nil {
		return nil, s.mapErrorToSessionError(err)
	} else if elem != nil {
		log.Debugf("RECV(%s): %v", s.id, elem)

		if atomic.LoadUint32(&s.started) == 0 {
			if err := s.validateStreamElement(elem); err != nil {
				return nil, err
			}
			if s.isInitiating {
				s.mu.Lock()
				s.streamID = elem.ID()
				s.mu.Unlock()
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

func (s *Session) setWriteDeadline(ctx context.Context) {
	d, ok := ctx.Deadline()
	if !ok {
		return
	}
	_ = s.tr.SetWriteDeadline(d)
}

func (s *Session) buildStanza(elem xmpp.XElement) (xmpp.Stanza, *Error) {
	if err := s.validateNamespace(elem); err != nil {
		return nil, err
	}
	fromJID, toJID, err := s.extractAddresses(elem)
	if err != nil {
		return nil, err
	}
	switch elem.Name() {
	case xmpp.IQName:
		iq, err := xmpp.NewIQFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, &Error{Element: elem, UnderlyingErr: xmpp.ErrBadRequest}
		}
		return iq, nil

	case xmpp.PresenceName:
		presence, err := xmpp.NewPresenceFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, &Error{Element: elem, UnderlyingErr: xmpp.ErrBadRequest}
		}
		return presence, nil

	case xmpp.MessageName:
		message, err := xmpp.NewMessageFromElement(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, &Error{Element: elem, UnderlyingErr: xmpp.ErrBadRequest}
		}
		return message, nil
	}
	return nil, &Error{UnderlyingErr: streamerror.ErrUnsupportedStanzaType}
}

func (s *Session) extractAddresses(elem xmpp.XElement) (*jid.JID, *jid.JID, *Error) {
	var fromJID, toJID *jid.JID
	var err error

	from := elem.From()
	if !s.isServer {
		// do not validate 'from' address until full user JID has been set
		if s.jid().IsFullWithUser() {
			if len(from) > 0 && !s.isValidFrom(from) {
				return nil, nil, &Error{UnderlyingErr: streamerror.ErrInvalidFrom}
			}
		}
		fromJID = s.jid()
	} else {
		j, err := jid.NewWithString(from, false)
		if err != nil || j.Domain() != s.remoteDomain {
			return nil, nil, &Error{UnderlyingErr: streamerror.ErrInvalidFrom}
		}
		fromJID = j
	}

	// validate 'to' address
	to := elem.To()
	if len(to) > 0 {
		toJID, err = jid.NewWithString(elem.To(), false)
		if err != nil {
			return nil, nil, &Error{Element: elem, UnderlyingErr: xmpp.ErrJidMalformed}
		}
	} else {
		toJID = s.jid().ToBareJID() // account's bare JID as default 'to'
	}
	return fromJID, toJID, nil
}

func (s *Session) isValidFrom(from string) bool {
	validFrom := false
	j, err := jid.NewWithString(from, false)
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

func (s *Session) validateStreamElement(elem xmpp.XElement) *Error {
	switch s.tr.Type() {
	case transport.Socket:
		if elem.Name() != "stream:stream" {
			return &Error{UnderlyingErr: streamerror.ErrUnsupportedStanzaType}
		}
		if elem.Namespace() != s.namespace() || elem.Attributes().Get("xmlns:stream") != streamNamespace {
			return &Error{UnderlyingErr: streamerror.ErrInvalidNamespace}
		}
	}
	to := elem.To()
	if len(to) > 0 && !s.hosts.IsLocalHost(to) {
		return &Error{UnderlyingErr: streamerror.ErrHostUnknown}
	}
	if elem.Version() != "1.0" {
		return &Error{UnderlyingErr: streamerror.ErrUnsupportedVersion}
	}
	return nil
}

func (s *Session) validateNamespace(elem xmpp.XElement) *Error {
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

func (s *Session) jid() *jid.JID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sJID
}

func (s *Session) mapErrorToSessionError(err error) *Error {
	switch err {
	case nil, io.EOF, io.ErrUnexpectedEOF:
		break

	case xmpp.ErrStreamClosedByPeer:
		_ = s.Close(context.Background())

	case xmpp.ErrTooLargeStanza:
		return &Error{UnderlyingErr: streamerror.ErrPolicyViolation}

	default:
		switch e := err.(type) {
		case net.Error:
			if e.Timeout() {
				return &Error{UnderlyingErr: streamerror.ErrConnectionTimeout}
			}
			return &Error{UnderlyingErr: err}

		case *stdxml.SyntaxError:
			return &Error{UnderlyingErr: streamerror.ErrInvalidXML}
		default:
			return &Error{UnderlyingErr: err}
		}
	}
	return &Error{}
}
