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

// Transport represents a stream transport mechanism.
type Transport interface {
	io.ReadWriteCloser

	// StartTLS secures the transport using SSL/TLS
	StartTLS(*tls.Config)

	// EnableCompression activates a compression
	// mechanism on the transport.
	EnableCompression(config.CompressionLevel)

	// ChannelBindingBytes returns current transport
	// channel binding bytes.
	ChannelBindingBytes(config.ChannelBindingMechanism) []byte
}
