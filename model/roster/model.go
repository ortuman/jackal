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

package rostermodel

import "github.com/jackal-xmpp/stravaganza"

const (
	// None represents 'none' subscription type.
	None = "none"

	// From represents 'from' subscription type.
	From = "from"

	// To represents 'to' subscription type.
	To = "to"

	// Both represents 'both' subscription type.
	Both = "both"

	// Remove represents 'remove' subscription type.
	Remove = "remove"
)

// Item represents a roster item entity.
type Item struct {
	Username     string
	JID          string
	Name         string
	Subscription string
	Ask          bool
	Groups       []string
}

// Notification represents a roster subscription pending notification.
type Notification struct {
	Contact  string
	JID      string
	Presence *stravaganza.Presence
}
