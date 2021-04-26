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

package transport

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"time"

	"github.com/ortuman/jackal/pkg/transport/compress"
	"golang.org/x/time/rate"
)

// Type represents a stream transport type.
type Type int

const (
	// Socket represents a socket transport type.
	Socket Type = iota + 1
)

// String returns TransportType string representation.
func (tt Type) String() string {
	switch tt {
	case Socket:
		return "socket"
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
	Type() Type

	// WriteString writes a raw string to the transport.
	WriteString(s string) (n int, err error)

	// Flush writes any buffered data to the underlying io.Writer.
	Flush() error

	// SetWriteDeadline sets the deadline for future write calls.
	SetWriteDeadline(d time.Time) error

	// SetReadRateLimiter sets transport read rate limiter.
	SetReadRateLimiter(rLim *rate.Limiter) error

	// StartTLS secures the transport using SSL/TLS
	StartTLS(cfg *tls.Config, asClient bool)

	// EnableCompression activates a compression mechanism on the transport.
	EnableCompression(compress.Level)

	// ChannelBindingBytes returns current transport channel binding bytes.
	ChannelBindingBytes(ChannelBindingMechanism) []byte

	// PeerCertificates returns the certificate chain presented by remote peer.
	PeerCertificates() []*x509.Certificate
}

type tlsStateQueryable interface {
	ConnectionState() tls.ConnectionState
}
