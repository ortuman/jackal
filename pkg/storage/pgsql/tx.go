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

package pgsqlrepository

import (
	"database/sql"

	"github.com/ortuman/jackal/pkg/storage/repository"
)

type repTx struct {
	repository.User
	repository.Last
	repository.Capabilities
	repository.Offline
	repository.BlockList
	repository.Private
	repository.Roster
	repository.VCard
	repository.Locker
}

func newRepTx(tx *sql.Tx) *repTx {
	return &repTx{
		User:         &pgSQLUserRep{conn: tx},
		Last:         &pgSQLLastRep{conn: tx},
		Capabilities: &pgSQLCapabilitiesRep{conn: tx},
		Offline:      &pgSQLOfflineRep{conn: tx},
		BlockList:    &pgSQLBlockListRep{conn: tx},
		Private:      &pgSQLPrivateRep{conn: tx},
		Roster:       &pgSQLRosterRep{conn: tx},
		VCard:        &pgSQLVCardRep{conn: tx},
		Locker:       &pgSQLLocker{conn: tx},
	}
}
