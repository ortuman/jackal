/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

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

type Callback interface {
	TransportReadBytes([]byte)
	TransportError(error)
}

type Transport struct {
	Callback Callback

	Write               func(b []byte)
	WriteAndWait        func(b []byte)
	Close               func()
	StartTLS            func() error
	EnableCompression   func(level CompressionLevel)
	ChannelBindingBytes func(mechanism ChannelBindingMechanism) []byte
}
