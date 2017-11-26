/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"crypto/tls"
	"errors"
)

// compression level
type CompressionLevel int

const (
	DefaultCompressionLevel = iota
	BestCompressionLevel
	SpeedCompressionLevel
)

// channel binding mechanisms
type ChannelBindingMechanism int

const (
	TLSUnique = iota
	TLSServerEndPoint
)

var (
	// ErrServerClosedTransport indicates that the underlying transport has been closed by server.
	ErrServerClosedTransport = errors.New("transport closed by server")

	// ErrRemotePeerClosedTransport indicates that the underlying transport has been closed by remote peer.
	ErrRemotePeerClosedTransport = errors.New("transport closed by remote peer")
)

type Transport struct {
	Write               func(b []byte)
	WriteAndWait        func(b []byte)
	Read                func() ([]byte, error)
	Close               func()
	StartTLS            func(*tls.Config) error
	EnableCompression   func(level CompressionLevel)
	ChannelBindingBytes func(mechanism ChannelBindingMechanism) []byte
}
