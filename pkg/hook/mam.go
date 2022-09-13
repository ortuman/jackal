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
	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
)

const (
	// ArchiveMessageQueried hook runs whenever an archive is queried.
	ArchiveMessageQueried = "mam.message.queried"

	// ArchiveMessageArchived hook runs whenever a message is archived.
	ArchiveMessageArchived = "mam.message.archieved"
)

// MamInfo contains all information associated to a mam (XEP-0313) event.
type MamInfo struct {
	// ArchiveID is the id of the mam archive associated to this event.
	ArchiveID string

	// Message is the message stanza associated to this event.
	Message *archivemodel.Message

	// Filters contains filters applied to the archive queried event.
	Filters *archivemodel.Filters
}
