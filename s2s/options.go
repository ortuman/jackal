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

import "time"

// Options defines S2S connection options.
type Options struct {
	DialTimeout time.Duration

	DialbackSecret string

	// ConnectTimeout defines connection timeout.
	ConnectTimeout time.Duration

	// KeepAlive defines stream read timeout.
	KeepAlive time.Duration

	// RequestTimeout defines S2S stream request timeout.
	RequestTimeout time.Duration

	// MaxStanzaSize is the maximum size a listener incoming stanza may have.
	MaxStanzaSize int
}
