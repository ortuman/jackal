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
	"github.com/jackal-xmpp/stravaganza/v2"
)

const (
	// S2SOutStreamRegistered event is posted when an outgoing S2S connection is registered.
	S2SOutStreamRegistered = "s2s.out.stream.registered"

	// S2SOutStreamUnregistered event is posted when an outgoing S2S connection is unregistered.
	S2SOutStreamUnregistered = "s2s.out.stream.unregistered"

	// S2SOutStreamElementSent event is posted whenever a XMPP element is sent over an outgoing S2S stream.
	S2SOutStreamElementSent = "s2s.out.stream.element_sent"

	// S2SInStreamRegistered event is posted when an incoming S2S connection is registered.
	S2SInStreamRegistered = "s2s.in.stream.registered"

	// S2SInStreamUnregistered event is posted when an incoming S2S connection is unregistered.
	S2SInStreamUnregistered = "s2s.in.stream.unregistered"

	// S2SInStreamElementReceived event is posted when a XMPP element is received over an incoming S2S stream.
	S2SInStreamElementReceived = "s2s.in.stream.stanza_received"

	// S2SInStreamIQReceived event is posted when an iq stanza is received over an incoming S2S stream.
	S2SInStreamIQReceived = "s2s.in.stream.iq_received"

	// S2SInStreamPresenceReceived event is posted when a presence stanza is received over an incoming S2S stream.
	S2SInStreamPresenceReceived = "s2s.in.stream.presence_received"

	// S2SInStreamMessageReceived event is posted when a message stanza is received over an incoming S2S stream.
	S2SInStreamMessageReceived = "s2s.in.stream.message_received"

	// S2SInStreamIQRouted event is posted when an iq stanza is successfully routed to one ore more S2S streams.
	S2SInStreamIQRouted = "s2s.in.stream.iq_routed"

	// S2SInStreamPresenceRouted event is posted when a presence stanza is successfully routed to one ore more S2S streams.
	S2SInStreamPresenceRouted = "s2s.in.stream.presence_routed"

	// S2SInStreamMessageRouted event is posted when a message stanza is successfully routed to one ore more S2S streams.
	S2SInStreamMessageRouted = "s2s.in.stream.message_routed"

	// S2SInStreamMessageUnrouted event is posted when a received message stanza could not be routed
	// because no destination available resource was found.
	S2SInStreamMessageUnrouted = "s2s.in.stream.message_unrouted"
)

// S2SStreamEventInfo contains all info associated to a S2S event.
type S2SStreamEventInfo struct {
	// ID is the event stream identifier.
	ID string

	// Sender is the S2S sender domain.
	Sender string

	// Target is the S2S target domain.
	Target string

	// Element is the event associated XMPP element.
	Element stravaganza.Element
}
