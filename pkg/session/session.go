// Copyright 2020 The jackal Authors
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

package session

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/host"
	xmppparser "github.com/ortuman/jackal/pkg/parser"
	"github.com/ortuman/jackal/pkg/transport"
	"github.com/ortuman/jackal/pkg/util/ratelimiter"
)

const envLogStanzas = "JACKAL_LOG_STANZAS"

var logStanzas bool

func init() {
	logStanzas = os.Getenv(envLogStanzas) == "on"
}

const (
	jabberClientNamespace    = "jabber:client"
	jabberServerNamespace    = "jabber:server"
	jabberComponentNamespace = "jabber:component:accept"
	streamNamespace          = "http://etherx.jabber.org/streams"
	dialbackNamespace        = "jabber:server:dialback"
)

var (
	errAlreadyOpened        = errors.New("session: already opened")
	errAlreadyClosed        = errors.New("session: already closed")
	errInvalidSessionType   = errors.New("session: invalid session type")
	errUnsupportedTransport = errors.New("session: unsupported transport type")
)

// Type represents session type.
type Type uint8

const (
	// C2SSession represents a C2S session type.
	C2SSession Type = iota

	// S2SSession represents a S2S session type
	S2SSession

	// ComponentSession represents a component session type.
	ComponentSession
)

// Config structure is used to establish XMPP session configuration.
type Config struct {

	// MaxStanzaSize defines the maximum stanza size that can be read from the session transport.
	MaxStanzaSize int

	// IsOut defines whether or not this is an initiating entity session.
	IsOut bool
}

// Session represents an XMPP session between two peers.
type Session struct {
	id     string
	typ    Type
	cfg    Config
	hosts  hosts
	tr     transport.Transport
	pr     xmppParser
	logger kitlog.Logger

	streamID string
	jd       jid.JID
	opened   bool
	started  bool
}

// New creates a new session instance.
func New(typ Type, identifier string, tr transport.Transport, hosts *host.Hosts, cfg Config, logger kitlog.Logger) *Session {
	ss := &Session{
		typ:    typ,
		id:     identifier,
		cfg:    cfg,
		hosts:  hosts,
		tr:     tr,
		pr:     getParser(tr, cfg.MaxStanzaSize),
		logger: logger,
	}
	if !ss.cfg.IsOut {
		ss.streamID = uuid.New().String()
	}
	return ss
}

// StreamID returns session stream identifier.
func (ss *Session) StreamID() string {
	return ss.streamID
}

// SetFromJID updates current session from JID.
func (ss *Session) SetFromJID(jd *jid.JID) {
	ss.jd = *jd
}

// OpenStream initializes a session session sending the proper XMPP payload.
func (ss *Session) OpenStream(ctx context.Context) error {
	if ss.typ != C2SSession && ss.typ != S2SSession {
		return errInvalidSessionType
	}
	if ss.opened {
		return errAlreadyOpened
	}
	var b *stravaganza.Builder

	buf := &strings.Builder{}
	switch ss.tr.Type() {
	case transport.Socket:
		b = stravaganza.NewBuilder("stream:stream")
		b.WithAttribute(stravaganza.Namespace, ss.namespace())
		b.WithAttribute(stravaganza.Version, "1.0")
		b.WithAttribute(stravaganza.StreamNamespace, streamNamespace)
		if ss.typ == S2SSession {
			b.WithAttribute("xmlns:db", dialbackNamespace)
		}
		buf.WriteString(`<?xml version='1.0'?>`)

	default:
		return errUnsupportedTransport
	}

	if ss.cfg.IsOut {
		b.WithAttribute(stravaganza.From, ss.hosts.DefaultHostName())
		b.WithAttribute(stravaganza.To, ss.jd.Domain())
	} else {
		b.WithAttribute(stravaganza.From, ss.jd.Domain())
		b.WithAttribute(stravaganza.ID, ss.streamID)
	}

	elem := b.Build()
	if err := elem.ToXML(buf, false); err != nil {
		return err
	}
	if err := ss.sendString(ctx, buf.String()); err != nil {
		return err
	}
	ss.opened = true
	return nil
}

// OpenComponent initializes a component session sending the proper XMPP payload.
func (ss *Session) OpenComponent(ctx context.Context) error {
	if ss.typ != ComponentSession {
		return errInvalidSessionType
	}
	if ss.opened {
		return errAlreadyOpened
	}
	buf := &strings.Builder{}

	elem := stravaganza.NewBuilder("stream:stream").
		WithAttribute(stravaganza.Namespace, ss.namespace()).
		WithAttribute(stravaganza.StreamNamespace, streamNamespace).
		WithAttribute(stravaganza.From, ss.jd.Domain()).
		WithAttribute(stravaganza.ID, ss.streamID).
		Build()

	buf.WriteString(`<?xml version="1.0"?>`)
	if err := elem.ToXML(buf, false); err != nil {
		return err
	}
	if err := ss.sendString(ctx, buf.String()); err != nil {
		return err
	}
	ss.opened = true
	return nil
}

// Close closes session sending the proper XMPP payload.
func (ss *Session) Close(ctx context.Context) error {
	if !ss.opened {
		return errAlreadyClosed
	}
	ss.setWriteDeadline(ctx)

	var outStr string

	switch ss.tr.Type() {
	case transport.Socket:
		outStr = "</stream:stream>"
	}
	if err := ss.sendString(ctx, outStr); err != nil {
		return err
	}
	ss.opened = false
	ss.started = false
	return nil
}

// Send writes an XML element to the underlying session transport.
func (ss *Session) Send(ctx context.Context, elem stravaganza.Element) error {
	if logStanzas {
		level.Debug(ss.logger).Log("msg", fmt.Sprintf("SND(%s): %v", ss.id, elem))
	}
	ss.setWriteDeadline(ctx)
	if err := elem.ToXML(ss.tr, true); err != nil {
		return err
	}
	return ss.tr.Flush()
}

// Receive returns next incoming session element.
func (ss *Session) Receive() (stravaganza.Element, error) {
	elem, err := ss.pr.Parse()
	if err != nil {
		return nil, mapErrorToSessionError(err)
	}
	switch {
	case elem != nil:
		if logStanzas {
			level.Debug(ss.logger).Log("msg", fmt.Sprintf("RCV(%s): %v", ss.id, elem))
		}
		if elem.Name() == "stream:error" {
			return nil, nil // ignore stream error incoming element
		}
	default:
		return nil, nil
	}
	if !ss.started {
		if err := ss.validateStreamElement(elem); err != nil {
			return nil, err
		}
		if ss.cfg.IsOut {
			ss.streamID = elem.Attribute(stravaganza.ID)
		}
		ss.started = true
		return elem, nil
	}
	if !stravaganza.IsStanza(elem) {
		return elem, nil
	}
	return ss.buildStanza(elem)
}

// Reset resets session internal state.
func (ss *Session) Reset(tr transport.Transport) error {
	if !ss.cfg.IsOut {
		ss.streamID = uuid.New().String()
	}
	ss.tr = tr
	ss.pr = getParser(tr, ss.cfg.MaxStanzaSize)
	ss.opened = false
	ss.started = false
	return nil
}

func (ss *Session) sendString(ctx context.Context, str string) error {
	if logStanzas {
		level.Debug(ss.logger).Log("msg", fmt.Sprintf("SND(%s): %v", ss.id, str))
	}
	ss.setWriteDeadline(ctx)
	_, err := ss.tr.WriteString(str)
	if err != nil {
		return err
	}
	return ss.tr.Flush()
}

func (ss *Session) validateStreamElement(elem stravaganza.Element) error {
	switch ss.tr.Type() {
	case transport.Socket:
		if elem.Name() != "stream:stream" {
			return streamerror.E(streamerror.UnsupportedStanzaType)
		}
		ns := elem.Attribute(stravaganza.Namespace)
		streamNs := elem.Attribute(stravaganza.StreamNamespace)
		if ns != ss.namespace() || streamNs != streamNamespace {
			return streamerror.E(streamerror.InvalidNamespace)
		}
	}
	if ss.typ == ComponentSession {
		return nil
	}
	to := elem.Attribute(stravaganza.To)
	if len(to) > 0 && !ss.hosts.IsLocalHost(to) {
		return streamerror.E(streamerror.HostUnknown)
	}
	if elem.Attribute(stravaganza.Version) != "1.0" {
		return streamerror.E(streamerror.UnsupportedVersion)
	}
	return nil
}

func (ss *Session) buildStanza(elem stravaganza.Element) (stravaganza.Stanza, error) {
	if err := ss.validateNamespace(elem); err != nil {
		return nil, err
	}
	fromJID, toJID, err := ss.extractAddresses(elem)
	if err != nil {
		return nil, err
	}
	sb := stravaganza.NewBuilderFromElement(elem).
		WithAttribute(stravaganza.From, fromJID.String()).
		WithAttribute(stravaganza.To, toJID.String()).
		WithoutAttribute(stravaganza.Namespace)

	switch elem.Name() {
	case "iq":
		iq, err := sb.BuildIQ()
		if err != nil {
			return nil, stanzaerror.E(stanzaerror.BadRequest, elem)
		}
		return iq, nil

	case "presence":
		presence, err := sb.BuildPresence()
		if err != nil {
			return nil, stanzaerror.E(stanzaerror.BadRequest, elem)
		}
		return presence, nil

	case "message":
		message, err := sb.BuildMessage()
		if err != nil {
			return nil, stanzaerror.E(stanzaerror.BadRequest, elem)
		}
		return message, nil
	}
	return nil, streamerror.E(streamerror.UnsupportedStanzaType)
}

func (ss *Session) validateNamespace(elem stravaganza.Element) error {
	ns := elem.Attribute(stravaganza.Namespace)
	if len(ns) == 0 || ns == ss.namespace() {
		return nil
	}
	return streamerror.E(streamerror.InvalidNamespace)
}

func (ss *Session) setWriteDeadline(ctx context.Context) {
	d, ok := ctx.Deadline()
	if !ok {
		return
	}
	_ = ss.tr.SetWriteDeadline(d)
}

func (ss *Session) namespace() string {
	switch ss.typ {
	case C2SSession:
		return jabberClientNamespace
	case S2SSession:
		return jabberServerNamespace
	case ComponentSession:
		return jabberComponentNamespace
	}
	return ""
}

func (ss *Session) extractAddresses(elem stravaganza.Element) (fromJID *jid.JID, toJID *jid.JID, err error) {
	from := elem.Attribute(stravaganza.From)
	switch ss.typ {
	case C2SSession:
		// do not validate 'from' address until full user JID has been set
		if ss.jd.IsFullWithUser() {
			if len(from) > 0 && !ss.isValidFrom(from) {
				return nil, nil, streamerror.E(streamerror.InvalidFrom)
			}
		}
		fromJID = &ss.jd

	default:
		j, err := jid.NewWithString(from, false)
		if err != nil || j.Domain() != ss.jd.Domain() {
			return nil, nil, streamerror.E(streamerror.InvalidFrom)
		}
		fromJID = j
	}

	// validate 'to' address
	to := elem.Attribute(stravaganza.To)
	if len(to) > 0 {
		toJID, err = jid.NewWithString(to, false)
		if err != nil {
			return nil, nil, stanzaerror.E(stanzaerror.JIDMalformed, elem)
		}
	} else {
		switch ss.typ {
		case C2SSession:
			toJID = ss.jd.ToBareJID() // account's bare JID as default 'to'
		default:
			toJID, _ = jid.NewWithString(ss.hosts.DefaultHostName(), true)
		}
	}
	return
}

func (ss *Session) isValidFrom(from string) bool {
	validFrom := false
	j, err := jid.NewWithString(from, false)
	if err == nil && j != nil {
		node := j.Node()
		domain := j.Domain()
		resource := j.Resource()

		validFrom = node == ss.jd.Node() && domain == ss.jd.Domain()
		if len(resource) > 0 {
			validFrom = validFrom && resource == ss.jd.Resource()
		}
	}
	return validFrom
}

func getParser(tr transport.Transport, maxStanzaSize int) *xmppparser.Parser {
	var pm xmppparser.ParsingMode
	switch tr.Type() {
	case transport.Socket:
		pm = xmppparser.SocketStream
	}
	return xmppparser.New(tr, pm, maxStanzaSize)
}

func mapErrorToSessionError(err error) error {
	switch err {
	case ratelimiter.ErrReadLimitExcedeed:
		se := streamerror.E(streamerror.PolicyViolation)
		se.Err = err
		se.ApplicationElement = stravaganza.NewBuilder("rate-limit-exceeded").
			WithAttribute(stravaganza.Namespace, "urn:xmpp:errors").
			Build()
		return se

	case xmppparser.ErrTooLargeStanza:
		se := streamerror.E(streamerror.PolicyViolation)
		se.Err = err
		se.ApplicationElement = stravaganza.NewBuilder("stanza-too-big").
			WithAttribute(stravaganza.Namespace, "urn:xmpp:errors").
			Build()
		return se

	default:
		switch err := err.(type) {
		case *xml.SyntaxError:
			se := streamerror.E(streamerror.InvalidXML)
			se.Err = err
			return se

		case net.Error:
			if !err.Timeout() {
				return err
			}
			se := streamerror.E(streamerror.ConnectionTimeout)
			se.Err = err
			return se

		default:
			return err // unmapped error
		}
	}
}
