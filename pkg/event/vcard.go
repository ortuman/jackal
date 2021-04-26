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

import "github.com/jackal-xmpp/stravaganza/v2"

const (
	// VCardFetched event is posted whenever a user vCard is fetched.
	VCardFetched = "vcard.fetched"

	// VCardUpdated event is posted whenever a user vCard is updated.
	VCardUpdated = "vcard.updated"
)

// VCardEventInfo contains all information associated to a vCard event.
type VCardEventInfo struct {
	// Username is the name of the vCard user associated to this event.
	Username string

	// VCard is the vCard element associated to this event.
	VCard stravaganza.Element
}
