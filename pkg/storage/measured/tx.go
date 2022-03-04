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

package measuredrepository

import "github.com/ortuman/jackal/pkg/storage/repository"

type measuredTx struct {
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

func newMeasuredTx(tx repository.Transaction) *measuredTx {
	return &measuredTx{
		User:         &measuredUserRep{rep: tx, inTx: true},
		Last:         &measuredLastRep{rep: tx, inTx: true},
		Capabilities: &measuredCapabilitiesRep{rep: tx, inTx: true},
		Offline:      &measuredOfflineRep{rep: tx, inTx: true},
		BlockList:    &measuredBlockListRep{rep: tx, inTx: true},
		Private:      &measuredPrivateRep{rep: tx, inTx: true},
		Roster:       &measuredRosterRep{rep: tx, inTx: true},
		VCard:        &measuredVCardRep{rep: tx, inTx: true},
		Locker:       &measuredLocker{rep: tx, inTx: true},
	}
}
