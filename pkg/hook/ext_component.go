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

import "github.com/jackal-xmpp/stravaganza"

const (
	// ExternalComponentRegistered hook runs when a external component connection is registered.
	ExternalComponentRegistered = "ext_component.stream.registered"

	// ExternalComponentUnregistered hook runs when a external component connection is unregistered.
	ExternalComponentUnregistered = "ext_component.stream.unregistered"

	// ExternalComponentElementReceived hook runs whenever an XMPP element is received over a external component stream.
	ExternalComponentElementReceived = "ext_component.stream.element_received"
)

// ExternalComponentInfo contains all info associated to an external component event.
type ExternalComponentInfo struct {
	// ID is the event stream identifier.
	ID string

	// Host is the external component host domain.
	Host string

	// Element is the event associated XMPP element.
	Element stravaganza.Element
}
