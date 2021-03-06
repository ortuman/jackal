// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package c2s

import (
	"crypto/tls"
	"time"

	"github.com/ortuman/jackal/pkg/transport/compress"
)

// ResourceConflict represents a resource conflict policy.
type ResourceConflict int8

const (
	// Override represents 'override' resource conflict policy.
	Override ResourceConflict = iota

	// Disallow represents 'disallow' resource conflict policy.
	Disallow

	// TerminateOld represents 'terminate_old' resource conflict policy.
	TerminateOld
)

// Config defines C2S connection configuration.
type Config struct {
	// ConnectTimeout defines connection timeout.
	ConnectTimeout time.Duration

	// AuthenticateTimeout defines authentication timeout.
	AuthenticateTimeout time.Duration

	// KeepAliveTimeout defines the maximum amount of time that an inactive connection
	// would be considered alive.
	KeepAliveTimeout time.Duration

	// RequestTimeout defines C2S stream request timeout.
	RequestTimeout time.Duration

	// MaxStanzaSize is the maximum size a listener incoming stanza may have.
	MaxStanzaSize int

	// CompressionLevel is the compression level that may be applied to the stream.
	// Valid values are 'default', 'best', 'speed' and 'no_compression'.
	CompressionLevel compress.Level

	// ResourceConflict defines the which rule should be applied in a resource conflict is detected.
	// Valid values are `override`, `disallow` and `terminate_old`.
	ResourceConflict ResourceConflict

	// UseTLS, if true, tls.Listen will be used as network listener.
	UseTLS bool

	// TLSConfig contains configuration to be used when TLS listener is enabled.
	TLSConfig *tls.Config
}
