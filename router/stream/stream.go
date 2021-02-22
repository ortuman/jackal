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

package stream

import (
	"context"
	"fmt"

	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
)

// C2SID type represents a C2S stream unique identifier string.
type C2SID uint64

// String returns C2S identifier string representation.
func (i C2SID) String() string {
	return fmt.Sprintf("c2s:%d", i)
}

// C2S represents a client-to-server XMPP stream.
type C2S interface {
	// ID returns C2S stream identifier.
	ID() C2SID

	// SetValue sets a stream context value.
	SetValue(ctx context.Context, k, val string) error

	// Value returns stream context value associated to cKey.
	Value(cKey string) string

	// JID returns stream associated jid or nil if none is set.
	JID() *jid.JID

	// Username returns stream associated username.
	Username() string

	// Domain returns stream associated domain.
	Domain() string

	// Resource returns stream associated resource.
	Resource() string

	// Presence returns stream associated presence stanza or nil if none is set.
	Presence() *stravaganza.Presence

	// SendElement writes element string representation to the underlying stream transport.
	SendElement(elem stravaganza.Element) <-chan error

	// Disconnect performs disconnection over the stream.
	Disconnect(streamErr *streamerror.Error) <-chan error
}

// S2SInID type represents an S2S stream unique identifier string.
type S2SInID uint64

// String returns S2SInID identifier string representation.
func (i S2SInID) String() string { return fmt.Sprintf("s2s:in:%d", i) }

// S2SIn represents an incoming server-to-server XMPP stream.
type S2SIn interface {
	// ID returns S2S incoming stream identifier.
	ID() S2SInID

	// Disconnect performs disconnection over the stream.
	Disconnect(streamErr *streamerror.Error) <-chan error
}

// S2SOutID type represents an S2S outgoing stream unique identifier string.
type S2SOutID struct {
	Sender string
	Target string
}

// String returns S2SOutID identifier string representation.
func (i S2SOutID) String() string { return fmt.Sprintf("s2s:out:%s-%s", i.Sender, i.Target) }

// S2SOut represents an outgoing server-to-server XMPP stream.
type S2SOut interface {
	// ID returns S2S outgoing stream identifier.
	ID() S2SOutID

	// SendElement writes element string representation to the underlying stream transport.
	SendElement(elem stravaganza.Element) <-chan error

	// Disconnect performs disconnection over the stream.
	Disconnect(streamErr *streamerror.Error) <-chan error
}

// DialbackResult represents S2S dialback result.
type DialbackResult struct {
	// Valid tells whether dialback validation was successfully completed.
	Valid bool

	// Error contains dialback resulting error element.
	// See https://xmpp.org/extensions/xep-0220.html#errors for more details.
	Error stravaganza.Element
}

// S2SDialback represents a server-to-server dialback XMPP stream.
type S2SDialback interface {
	// ID returns S2S outgoing stream identifier.
	ID() S2SOutID

	// Done returns a channel that's signaled when S2S dialback result.
	Done() <-chan DialbackResult

	// Disconnect performs disconnection over the stream.
	Disconnect(streamErr *streamerror.Error) <-chan error
}
