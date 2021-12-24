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

package s2s

import (
	"time"
)

// ListenersConfig defines a set of S2S listener configurations.
type ListenersConfig []ListenerConfig

// ListenerConfig defines S2S listener configuration.
type ListenerConfig struct {
	// BindAddr defines listener incoming connections address.
	BindAddr string `fig:"bind_addr"`

	// Port defines listener incoming connections port.
	Port int `fig:"port" default:"5269"`

	// ConnectTimeout defines connection timeout.
	ConnectTimeout time.Duration `fig:"conn_timeout" default:"3s"`

	// KeepAliveTimeout defines stream read timeout.
	KeepAliveTimeout time.Duration `fig:"keep_alive_timeout" default:"10m"`

	// RequestTimeout defines S2S stream request timeout.
	RequestTimeout time.Duration `fig:"req_timeout" default:"15s"`

	// MaxStanzaSize is the maximum size a listener incoming stanza may have.
	MaxStanzaSize int `fig:"max_stanza_size" default:"131072"`

	// DirectTLS, if true, tls.Listen will be used as network listener.
	DirectTLS bool `fig:"direct_tls"`
}

// OutConfig defines S2S out configuration.
type OutConfig struct {
	// DialbackSecret defines S2S dialback secret key.
	DialbackSecret string `fig:"dialback_secret"`

	// DialTimeout defines S2S out dialer timeout.
	DialTimeout time.Duration `fig:"dial_timeout" default:"5s"`

	// KeepAliveTimeout defines stream read timeout.
	KeepAliveTimeout time.Duration `fig:"keep_alive_timeout" default:"10m"`

	// RequestTimeout defines S2S stream request timeout.
	RequestTimeout time.Duration `fig:"req_timeout" default:"15s"`

	// MaxStanzaSize is the maximum size a listener incoming stanza may have.
	MaxStanzaSize int `fig:"max_stanza_size" default:"131072"`
}
