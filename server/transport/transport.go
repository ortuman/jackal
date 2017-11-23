/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import "io"

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
	SentBytes(int)
	StartedTLS()
	FailedStartTLS(error)
	Error(error)
}

type Transport struct {
	Callback  Callback
	KeepAlive int

	Write               func(b io.Reader)
	WriteAndWait        func(b io.Reader)
	Close               func()
	StartTLS            func()
	EnableCompression   func(level int)
	ChannelBindingBytes func(mechanism int) []byte
}
