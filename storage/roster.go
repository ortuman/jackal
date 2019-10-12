/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import rostermodel "github.com/ortuman/jackal/model/roster"

// rosterStorage defines storage operations for user's roster
type rosterStorage interface {
	UpsertRosterItem(ri *rostermodel.Item) (rostermodel.Version, error)
	DeleteRosterItem(username, jid string) (rostermodel.Version, error)
	FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error)
	FetchRosterItemsInGroups(username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error)
	FetchRosterItem(username, jid string) (*rostermodel.Item, error)
	UpsertRosterNotification(rn *rostermodel.Notification) error
	DeleteRosterNotification(contact, jid string) error
	FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error)
	FetchRosterNotifications(contact string) ([]rostermodel.Notification, error)
	FetchRosterGroups(username string) ([]string, error)
}

// UpsertRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func UpsertRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
	return instance().UpsertRosterItem(ri)
}

// DeleteRosterItem deletes a roster item entity from storage.
func DeleteRosterItem(username, jid string) (rostermodel.Version, error) {
	return instance().DeleteRosterItem(username, jid)
}

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error) {
	return instance().FetchRosterItems(username)
}

// FetchRosterItemsInGroups retrieves from storage all roster item entities
// associated to a given user and a set of groups.
func FetchRosterItemsInGroups(username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	return instance().FetchRosterItemsInGroups(username, groups)
}

// FetchRosterItem retrieves from storage a roster item entity.
func FetchRosterItem(username, jid string) (*rostermodel.Item, error) {
	return instance().FetchRosterItem(username, jid)
}

// UpsertRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func UpsertRosterNotification(rn *rostermodel.Notification) error {
	return instance().UpsertRosterNotification(rn)
}

// DeleteRosterNotification deletes a roster notification entity from storage.
func DeleteRosterNotification(contact, jid string) error {
	return instance().DeleteRosterNotification(contact, jid)
}

// FetchRosterNotification retrieves from storage a roster notification entity.
func FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error) {
	return instance().FetchRosterNotification(contact, jid)
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func FetchRosterNotifications(contact string) ([]rostermodel.Notification, error) {
	return instance().FetchRosterNotifications(contact)
}

// FetchRosterGroups retrieves all groups associated to a user roster
func FetchRosterGroups(username string) ([]string, error) {
	return instance().FetchRosterGroups(username)
}
