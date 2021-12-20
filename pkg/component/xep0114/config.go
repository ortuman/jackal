// Copyright 2021 The jackal Authors
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

package xep0114

import "time"

// ListenersConfig defines a set of component listener configurations.
type ListenersConfig []ListenerConfig

// ListenerConfig defines component connection configuration.
type ListenerConfig struct {
	// BindAddr defines listener incoming connections address.
	BindAddr string `fig:"bind_addr"`

	// Port defines listener incoming connections port.
	Port int `fig:"port" default:"5275"`

	// ConnectTimeout defines connection timeout.
	ConnectTimeout time.Duration

	// KeepAliveTimeout defines the maximum amount of time that an inactive connection
	// would be considered alive.
	KeepAliveTimeout time.Duration

	// RequestTimeout defines component stream request timeout.
	RequestTimeout time.Duration

	// MaxStanzaSize is the maximum size a listener incoming stanza may have.
	MaxStanzaSize int

	// Secret is the external component's shared secret.
	Secret string
}
