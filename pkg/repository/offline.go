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

package repository

import (
	"context"

	"github.com/jackal-xmpp/stravaganza/v2"
)

// Offline defines user offline repository operations.
type Offline interface {
	// InsertOfflineMessage inserts a new message element into user's offline queue.
	InsertOfflineMessage(ctx context.Context, message *stravaganza.Message, username string) error

	// CountOfflineMessages returns current length of user's offline queue.
	CountOfflineMessages(ctx context.Context, username string) (int, error)

	// FetchOfflineMessages retrieves from repository current user offline queue.
	FetchOfflineMessages(ctx context.Context, username string) ([]*stravaganza.Message, error)

	// DeleteOfflineMessages clears a user offline queue.
	DeleteOfflineMessages(ctx context.Context, username string) error
}
