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

package hook

import (
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
)

const (
	// C2SStreamRegistered event is posted when a C2S connection is registered.
	C2SStreamRegistered = "c2s.stream.registered"

	// C2SStreamBinded event is posted when C2S stream is bounded.
	C2SStreamBinded = "c2s.stream.binded"

	// C2SStreamUnregistered event is posted when a C2S connection is unregistered.
	C2SStreamUnregistered = "c2s.stream.unregistered"

	// C2SStreamElementReceived event is posted when a XMPP element is received over a C2S stream.
	C2SStreamElementReceived = "c2s.stream.element_received"

	// C2SStreamIQReceived event is posted when an iq stanza is received over a C2S stream.
	C2SStreamIQReceived = "c2s.stream.iq_received"

	// C2SStreamPresenceReceived event is posted when a presence stanza is received over a C2S stream.
	C2SStreamPresenceReceived = "c2s.stream.presence_received"

	// C2SStreamMessageReceived event is posted when a message stanza is received over a C2S stream.
	C2SStreamMessageReceived = "c2s.stream.message_received"

	// C2SStreamWillRouteElement event is posted when an XMPP element is about to be routed over a C2S stream.
	C2SStreamWillRouteElement = "c2s.stream.will_route_element"

	// C2SStreamIQRouted event is posted when an iq stanza is successfully routed to one ore more C2S streams.
	C2SStreamIQRouted = "c2s.stream.iq_routed"

	// C2SStreamPresenceRouted event is posted when a presence stanza is successfully routed to one ore more C2S streams.
	C2SStreamPresenceRouted = "c2s.stream.presence_routed"

	// C2SStreamMessageRouted event is posted when a message stanza is successfully routed to one ore more C2S streams.
	C2SStreamMessageRouted = "c2s.stream.message_routed"

	// C2SStreamElementSent event is posted when a XMPP element is sent over a C2S stream.
	C2SStreamElementSent = "c2s.stream.element_sent"
)

// C2SStreamInfo contains all info associated to a C2S stream event.
type C2SStreamInfo struct {
	// ID is the event stream identifier.
	ID string

	// JID represents the event associated JID.
	JID *jid.JID

	// Targets contains all JIDs to which the event stanza was routed.
	Targets []jid.JID

	// Element is the event associated XMPP element.
	Element stravaganza.Element
}
