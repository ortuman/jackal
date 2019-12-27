/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	rostermodel "github.com/ortuman/jackal/model/roster"
)

// rosterStorage defines storage operations for user's roster
type rosterStorage interface {
	UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) (rostermodel.Version, error)
	DeleteRosterItem(ctx context.Context, username, jid string) (rostermodel.Version, error)
	FetchRosterItems(ctx context.Context, username string) ([]rostermodel.Item, rostermodel.Version, error)
	FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error)
	FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error)
	UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error
	DeleteRosterNotification(ctx context.Context, contact, jid string) error
	FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error)
	FetchRosterNotifications(ctx context.Context, contact string) ([]rostermodel.Notification, error)
	FetchRosterGroups(ctx context.Context, username string) ([]string, error)
}

// UpsertRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) (rostermodel.Version, error) {
	return instance().UpsertRosterItem(ctx, ri)
}

// DeleteRosterItem deletes a roster item entity from storage.
func DeleteRosterItem(ctx context.Context, username, jid string) (rostermodel.Version, error) {
	return instance().DeleteRosterItem(ctx, username, jid)
}

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func FetchRosterItems(ctx context.Context, username string) ([]rostermodel.Item, rostermodel.Version, error) {
	return instance().FetchRosterItems(ctx, username)
}

// FetchRosterItemsInGroups retrieves from storage all roster item entities
// associated to a given user and a set of groups.
func FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	return instance().FetchRosterItemsInGroups(ctx, username, groups)
}

// FetchRosterItem retrieves from storage a roster item entity.
func FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error) {
	return instance().FetchRosterItem(ctx, username, jid)
}

// UpsertRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error {
	return instance().UpsertRosterNotification(ctx, rn)
}

// DeleteRosterNotification deletes a roster notification entity from storage.
func DeleteRosterNotification(ctx context.Context, contact, jid string) error {
	return instance().DeleteRosterNotification(ctx, contact, jid)
}

// FetchRosterNotification retrieves from storage a roster notification entity.
func FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	return instance().FetchRosterNotification(ctx, contact, jid)
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func FetchRosterNotifications(ctx context.Context, contact string) ([]rostermodel.Notification, error) {
	return instance().FetchRosterNotifications(ctx, contact)
}

// FetchRosterGroups retrieves all groups associated to a user roster
func FetchRosterGroups(ctx context.Context, username string) ([]string, error) {
	return instance().FetchRosterGroups(ctx, username)
}
