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

	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"

	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
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

	// SetInfoValue sets a C2S stream info value.
	SetInfoValue(ctx context.Context, k string, val interface{}) error

	// Info returns C2S stream context.
	Info() c2smodel.Info

	// JID returns stream associated jid or nil if none is set.
	JID() *jid.JID

	// Username returns stream associated username.
	Username() string

	// Domain returns stream associated domain.
	Domain() string

	// Resource returns stream associated resource.
	Resource() string

	// IsSecured returns whether or not the XMPP stream has been secured using SSL/TLS.
	IsSecured() bool

	// IsAuthenticated returns whether or not the XMPP stream has successfully authenticated.
	IsAuthenticated() bool

	// IsBinded returns whether or not the XMPP stream has completed resource binding.
	IsBinded() bool

	// Presence returns stream associated presence stanza or nil if none is set.
	Presence() *stravaganza.Presence

	// SendElement writes element string representation to the underlying stream transport.
	SendElement(elem stravaganza.Element) <-chan error

	// Disconnect performs disconnection over the stream.
	Disconnect(streamErr *streamerror.Error) <-chan error

	// Resume resumes a previously initiated c2s session.
	Resume(jd *jid.JID, pr *stravaganza.Presence, inf c2smodel.Info)

	// Done returns a channel that's closed when stream transport and all associated resources have been released.
	Done() <-chan struct{}
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

	// Done returns a channel that's closed when stream transport and all associated resources have been released.
	Done() <-chan struct{}
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

	// Disconnect performs disconnection over the stream.
	Disconnect(streamErr *streamerror.Error) <-chan error

	// DialbackResult returns a channel that's signaled when S2S dialback result.
	DialbackResult() <-chan DialbackResult
}
