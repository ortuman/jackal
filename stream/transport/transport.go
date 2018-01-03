/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"crypto/tls"
	"io"

	"github.com/ortuman/jackal/config"
)

type Transport interface {
	io.ReadWriteCloser

	StartTLS(*tls.Config)
	EnableCompression(config.CompressionLevel)
	ChannelBindingBytes(config.ChannelBindingMechanism) []byte
}
