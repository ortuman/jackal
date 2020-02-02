/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// InStream represents a generic incoming stream.
type InStream interface {
	ID() string
	Disconnect(ctx context.Context, err error)
}

// InOutStream represents a generic incoming/outgoing stream.
type InOutStream interface {
	InStream
	SendElement(ctx context.Context, elem xmpp.XElement)
}

// C2S represents a client-to-server XMPP stream.
type C2S interface {
	InOutStream

	Context() context.Context

	GetContextValue(key interface{}) interface{}
	SetContextValue(key, value interface{})

	Username() string
	Domain() string
	Resource() string

	JID() *jid.JID

	IsSecured() bool
	IsAuthenticated() bool

	Presence() *xmpp.Presence
}

// S2SIn represents an incoming server-to-server XMPP stream.
type S2SIn interface {
	InStream
}

// S2SOut represents an outgoing server-to-server XMPP stream.
type S2SOut interface {
	InOutStream
}
