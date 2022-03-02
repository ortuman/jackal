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

package boltdb

import (
	"github.com/ortuman/jackal/pkg/storage/repository"
	bolt "go.etcd.io/bbolt"
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

func newRepTx(tx *bolt.Tx) *repTx {
	return &repTx{
		User:         newUserRep(tx),
		Last:         newLastRep(tx),
		Capabilities: newCapsRep(tx),
		Offline:      newOfflineRep(tx),
		BlockList:    newBlockListRep(tx),
		Private:      newPrivateRep(tx),
		Roster:       newRosterRep(tx),
		VCard:        newVCardRep(tx),
		Locker:       newLockerRep(),
	}
}
