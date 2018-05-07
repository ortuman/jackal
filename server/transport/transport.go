/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"crypto/tls"
	"errors"
	"io"

	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/xml"
)

// ErrTooLargeStanza is returned by ReadElement when the size of
// the received stanza is too large.
var ErrTooLargeStanza = errors.New("too large stanza")

// TransportType represents a stream transport type (socket).
type TransportType int

const (
	// Socket represents a socket transport type.
	Socket TransportType = iota + 1

	// WebSocket represents a websocket transport type.
	WebSocket
)

// String returns TransportType string representation.
func (tt TransportType) String() string {
	switch tt {
	case Socket:
		return "socket"
	case WebSocket:
		return "websocket"
	}
	return ""
}

// ChannelBindingMechanism represents a scram channel binding mechanism.
type ChannelBindingMechanism int

const (
	// TLSUnique represents 'tls-unique' channel binding mechanism.
	TLSUnique ChannelBindingMechanism = iota
)

// Transport represents a stream transport mechanism.
type Transport interface {
	io.Closer

	// ReadElement reads next available XML element.
	ReadElement() (xml.XElement, error)

	// WriteString writes a raw string to the transport.
	WriteString(string) error

	// WriteElement writes an element to the transport
	// serializing it to it's XML representation.
	WriteElement(elem xml.XElement, includeClosing bool) error

	// StartTLS secures the transport using SSL/TLS
	StartTLS(*tls.Config)

	// EnableCompression activates a compression
	// mechanism on the transport.
	EnableCompression(compress.Level)

	// ChannelBindingBytes returns current transport
	// channel binding bytes.
	ChannelBindingBytes(ChannelBindingMechanism) []byte
}
