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

type Transport interface {
	Write(b io.Reader)
	WriteAndWait(b io.Reader)

	Close()

	StartTLS()
	EnableCompression(level int)

	ChannelBindingBytes(mechanism int) []byte
}
