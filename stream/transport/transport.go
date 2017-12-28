/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"crypto/tls"
	"errors"

	"github.com/ortuman/jackal/config"
)

// channel binding mechanisms
const (
	TLSUnique = "tls-unique"
)

var (
	// ErrServerClosedTransport indicates that the underlying transport has been closed by server.
	ErrServerClosedTransport = errors.New("transport closed by server")

	// ErrRemotePeerClosedTransport indicates that the underlying transport has been closed by remote peer.
	ErrRemotePeerClosedTransport = errors.New("transport closed by remote peer")
)

type Transport interface {
	Write(p []byte) (n int, err error)
	Read(p []byte) (n int, err error)
	Close() error

	StartTLS(*tls.Config)
	EnableCompression(config.CompressionLevel)
	ChannelBindingBytes(string) []byte
}
