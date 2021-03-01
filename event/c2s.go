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

package event

import (
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
)

const (
	// C2SStreamRegistered event is posted when a C2S connection is registered.
	C2SStreamRegistered = "c2s.stream.registered"

	// C2SStreamBounded event is posted when C2S stream is bounded.
	C2SStreamBounded = "c2s.stream.bounded"

	// C2SStreamUnregistered event is posted when a C2S connection is unregistered.
	C2SStreamUnregistered = "c2s.stream.unregistered"

	// C2SStreamStanzaReceived event is posted whenever a stanza is received over a C2S stream.
	C2SStreamStanzaReceived = "c2s.stream.stanza_received"

	// C2SStreamIQReceived event is posted whenever an iq stanza is received over a C2S stream.
	C2SStreamIQReceived = "c2s.stream.iq_received"

	// C2SStreamPresenceReceived event is posted whenever a presence stanza is received over a C2S stream.
	C2SStreamPresenceReceived = "c2s.stream.presence_received"

	// C2SStreamMessageReceived event is posted whenever a message stanza is received over a C2S stream.
	C2SStreamMessageReceived = "c2s.stream.message_received"

	// C2SStreamMessageUnrouted event is posted whenever a previously received message stanza could not be routed
	// because no destination available resource was found.
	C2SStreamMessageUnrouted = "c2s.stream.message_unrouted"
)

// C2SStreamEventInfo contains all info associated to a C2S stream event.
type C2SStreamEventInfo struct {
	// ID is the event stream identifier.
	ID string

	// JID represents the event associated JID.
	JID *jid.JID

	// Stanza represents the event associated stanza.
	Stanza stravaganza.Stanza
}