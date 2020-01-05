/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	rostermodel "github.com/ortuman/jackal/model/roster"
)

// Roster defines storage operations for user's roster.
type Roster interface {
	// UpsertRosterItem inserts a new roster item entity into storage,
	// or updates it in case it's been previously inserted.
	UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) (rostermodel.Version, error)

	// DeleteRosterItem deletes a roster item entity from storage.
	DeleteRosterItem(ctx context.Context, username, jid string) (rostermodel.Version, error)

	// FetchRosterItems retrieves from storage all roster item entities
	// associated to a given user.
	FetchRosterItems(ctx context.Context, username string) ([]rostermodel.Item, rostermodel.Version, error)

	// FetchRosterItemsInGroups retrieves from storage all roster item entities
	// associated to a given user and a set of groups.
	FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error)

	// FetchRosterItem retrieves from storage a roster item entity.
	FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error)

	// UpsertRosterNotification inserts a new roster notification entity
	// into storage, or updates it in case it's been previously inserted.
	UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error

	// DeleteRosterNotification deletes a roster notification entity from storage.
	DeleteRosterNotification(ctx context.Context, contact, jid string) error

	// FetchRosterNotification retrieves from storage a roster notification entity.
	FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error)

	// FetchRosterNotifications retrieves from storage all roster notifications
	// associated to a given user.
	FetchRosterNotifications(ctx context.Context, contact string) ([]rostermodel.Notification, error)

	// FetchRosterGroups retrieves all groups associated to a user roster.
	FetchRosterGroups(ctx context.Context, username string) ([]string, error)
}
