/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import "crypto/tls"

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

type Callback struct {
	ReadBytes func([]byte)
	Close     func()
	Error     func(error)
}

type Transport struct {
	Write               func(b []byte)
	WriteAndWait        func(b []byte)
	Close               func()
	StartTLS            func(*tls.Config) error
	EnableCompression   func(level CompressionLevel)
	ChannelBindingBytes func(mechanism ChannelBindingMechanism) []byte
}
