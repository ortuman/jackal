// Copyright 2021 The jackal Authors
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
	"github.com/jackal-xmpp/stravaganza/jid"
)

const (
	// BlockListFetched hook runs when a user block list is fetched.
	BlockListFetched = "blocklist.items.fetched"

	// BlockListItemsBlocked hook runs when one or more JIDs are blocked.
	BlockListItemsBlocked = "blocklist.items.blocked"

	// BlockListItemsUnblocked hook runs when one or more JIDs are unblocked.
	BlockListItemsUnblocked = "blocklist.items.unblocked"
)

// BlockListInfo contains all information associated to a blocklist event.
type BlockListInfo struct {
	// Username is the name of the user associated to this event.
	Username string

	// JIDs contains all JIDs associated to this event.
	JIDs []jid.JID
}
