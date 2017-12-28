/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"crypto/tls"

	"github.com/ortuman/jackal/config"
)

// channel binding mechanisms
const (
	TLSUnique = "tls-unique"
)

type Transport interface {
	Write(p []byte) (n int, err error)
	Read(p []byte) (n int, err error)
	Close() error

	StartTLS(*tls.Config)
	EnableCompression(config.CompressionLevel)
	ChannelBindingBytes(string) []byte
}
