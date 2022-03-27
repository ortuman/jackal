// Copyright 2022 The jackal Authors
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

package hook

import (
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
)

const (
	// C2SStreamConnected hook runs when a C2S connection is registered.
	C2SStreamConnected = "c2s.stream.connected"

	// C2SStreamBinded hook runs when C2S stream is bounded.
	C2SStreamBinded = "c2s.stream.binded"

	// C2SStreamDisconnected hook runs when a C2S connection is unregistered.
	C2SStreamDisconnected = "c2s.stream.disconnected"

	// C2SStreamTerminated hook runs when a C2S connection is terminated.
	C2SStreamTerminated = "c2s.stream.terminated"

	// C2SStreamElementReceived hook runs when a XMPP element is received over a C2S stream.
	C2SStreamElementReceived = "c2s.stream.element_received"

	// C2SStreamIQReceived hook runs when an iq stanza is received over a C2S stream.
	C2SStreamIQReceived = "c2s.stream.iq_received"

	// C2SStreamPresenceReceived hook runs when a presence stanza is received over a C2S stream.
	C2SStreamPresenceReceived = "c2s.stream.presence_received"

	// C2SStreamMessageReceived hook runs when a message stanza is received over a C2S stream.
	C2SStreamMessageReceived = "c2s.stream.message_received"

	// C2SStreamWillRouteElement hook runs when an XMPP element is about to be routed over a C2S stream.
	C2SStreamWillRouteElement = "c2s.stream.will_route_element"

	// C2SStreamIQRouted hook runs when an iq stanza is successfully routed to one ore more C2S streams.
	C2SStreamIQRouted = "c2s.stream.iq_routed"

	// C2SStreamPresenceRouted hook runs when a presence stanza is successfully routed to one ore more C2S streams.
	C2SStreamPresenceRouted = "c2s.stream.presence_routed"

	// C2SStreamMessageRouted hook runs when a message stanza is successfully routed to one ore more C2S streams.
	C2SStreamMessageRouted = "c2s.stream.message_routed"

	// C2SStreamElementSent hook runs when a XMPP element is sent over a C2S stream.
	C2SStreamElementSent = "c2s.stream.element_sent"
)

// C2SStreamInfo contains all info associated to a C2S stream event.
type C2SStreamInfo struct {
	// ID is the event stream identifier.
	ID string

	// JID represents the event associated JID.
	JID *jid.JID

	// Presence is current C2S resource presence.
	Presence *stravaganza.Presence

	// Element is the event associated XMPP element.
	Element stravaganza.Element

	// Targets contains all JIDs to which event stanza was routed.
	Targets []jid.JID

	// DisconnectError contains the original error that caused stream disconnection.
	DisconnectError error
}
