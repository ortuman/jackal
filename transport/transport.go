/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"crypto/tls"
	"crypto/x509"
	"io"

	"github.com/ortuman/jackal/transport/compress"
)

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
	io.ReadWriteCloser

	// Type returns transport type value.
	Type() TransportType

	// WriteString writes a raw string to the transport.
	WriteString(s string) (n int, err error)

	// StartTLS secures the transport using SSL/TLS
	StartTLS(cfg *tls.Config, asClient bool)

	// EnableCompression activates a compression
	// mechanism on the transport.
	EnableCompression(compress.Level)

	// ChannelBindingBytes returns current transport
	// channel binding bytes.
	ChannelBindingBytes(ChannelBindingMechanism) []byte

	// PeerCertificates returns the certificate chain
	// presented by remote peer.
	PeerCertificates() []*x509.Certificate
}
