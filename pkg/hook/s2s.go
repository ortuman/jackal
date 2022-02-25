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
	"github.com/jackal-xmpp/stravaganza"
)

const (
	// S2SOutStreamConnected hook runs when an outgoing S2S connection is registered.
	S2SOutStreamConnected = "s2s.out.stream.connected"

	// S2SOutStreamDisconnected hook runs when an outgoing S2S connection is unregistered.
	S2SOutStreamDisconnected = "s2s.out.stream.disconnected"

	// S2SOutStreamElementSent hook runs whenever a XMPP element is sent over an outgoing S2S stream.
	S2SOutStreamElementSent = "s2s.out.stream.element_sent"

	// S2SInStreamRegistered hook runs when an incoming S2S connection is registered.
	S2SInStreamRegistered = "s2s.in.stream.registered"

	// S2SInStreamUnregistered hook runs when an incoming S2S connection is unregistered.
	S2SInStreamUnregistered = "s2s.in.stream.unregistered"

	// S2SInStreamElementReceived hook runs when a XMPP element is received over an incoming S2S stream.
	S2SInStreamElementReceived = "s2s.in.stream.stanza_received"

	// S2SInStreamIQReceived hook runs when an iq stanza is received over an incoming S2S stream.
	S2SInStreamIQReceived = "s2s.in.stream.iq_received"

	// S2SInStreamPresenceReceived hook runs when a presence stanza is received over an incoming S2S stream.
	S2SInStreamPresenceReceived = "s2s.in.stream.presence_received"

	// S2SInStreamMessageReceived hook runs when a message stanza is received over an incoming S2S stream.
	S2SInStreamMessageReceived = "s2s.in.stream.message_received"

	// S2SInStreamWillRouteElement hook runs when an XMPP element is about to be routed on an incoming S2S stream.
	S2SInStreamWillRouteElement = "s2s.in.stream.will_route_element"

	// S2SInStreamIQRouted hook runs when an iq stanza is successfully routed to one ore more S2S streams.
	S2SInStreamIQRouted = "s2s.in.stream.iq_routed"

	// S2SInStreamPresenceRouted hook runs when a presence stanza is successfully routed to one ore more S2S streams.
	S2SInStreamPresenceRouted = "s2s.in.stream.presence_routed"

	// S2SInStreamMessageRouted hook runs when a message stanza is successfully routed to one ore more S2S streams.
	S2SInStreamMessageRouted = "s2s.in.stream.message_routed"
)

// S2SStreamInfo contains all info associated to a S2S event.
type S2SStreamInfo struct {
	// ID is the event stream identifier.
	ID string

	// Sender is the S2S sender domain.
	Sender string

	// Target is the S2S target domain.
	Target string

	// Element is the event associated XMPP element.
	Element stravaganza.Element
}
