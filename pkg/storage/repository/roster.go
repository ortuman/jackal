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

	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
)

// Roster defines user roster repository operations.
type Roster interface {
	// TouchRosterVersion increments user roster version.
	TouchRosterVersion(ctx context.Context, username string) (int, error)

	// FetchRosterVersion fetches user roster version.
	FetchRosterVersion(ctx context.Context, username string) (int, error)

	// UpsertRosterItem inserts a new roster item entity into repository.
	UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) error

	// DeleteRosterItem deletes a roster item entity from repository.
	DeleteRosterItem(ctx context.Context, username, jid string) error

	// DeleteRosterItems deletes all user roster items.
	DeleteRosterItems(ctx context.Context, username string) error

	// FetchRosterItems fetches from storage all roster item entities associated to a given user.
	FetchRosterItems(ctx context.Context, username string) ([]*rostermodel.Item, error)

	// FetchRosterItemsInGroups fetches from repository all roster item entities associated to a given user and a set of groups.
	FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]*rostermodel.Item, error)

	// FetchRosterItem fetches from repository a roster item entity.
	FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error)

	// UpsertRosterNotification inserts or updates a roster notification entity into repository.
	UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error

	// DeleteRosterNotification deletes a roster notification entity from repository.
	DeleteRosterNotification(ctx context.Context, contact, jid string) error

	// DeleteRosterNotifications deletes all contact roster notifications.
	DeleteRosterNotifications(ctx context.Context, contact string) error

	// FetchRosterNotification fetches from repository a roster notification entity.
	FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error)

	// FetchRosterNotifications fetches from repository all roster notifications associated to a user.
	FetchRosterNotifications(ctx context.Context, contact string) ([]*rostermodel.Notification, error)

	// FetchRosterGroups fetches all groups associated to a user roster.
	FetchRosterGroups(ctx context.Context, username string) ([]string, error)
}
