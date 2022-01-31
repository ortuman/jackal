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

package cachedrepository

import (
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type cachedTx struct {
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

func newCacheTx(c Cache, tx repository.Transaction) *cachedTx {
	return &cachedTx{
		User:         &cachedUserRep{c: c, rep: tx},
		Last:         tx,
		Capabilities: tx,
		Offline:      tx,
		BlockList:    tx,
		Private:      tx,
		Roster:       tx,
		VCard:        &cachedVCardRep{c: c, rep: tx},
		Locker:       tx,
	}
}
