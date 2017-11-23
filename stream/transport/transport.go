/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

// compression level
const (
	DefaultCompressionLevel = iota
	BestCompressionLevel
	SpeedCompressionLevel
)

// channel binding mechanisms
const (
	TLSUnique = iota
	TLSServerEndPoint
)

type Callback interface {
	ReadBytes([]byte)
	SentBytes([]byte)
	StartedTLS()
	FailedStartTLS(error)
	Error(error)
}

type Transport struct {
	Callback Callback

	Write               func(b []byte)
	WriteAndWait        func(b []byte)
	Close               func()
	StartTLS            func()
	EnableCompression   func(level int)
	ChannelBindingBytes func(mechanism int) []byte
}
