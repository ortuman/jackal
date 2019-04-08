package storage

import rostermodel "github.com/ortuman/jackal/model/roster"

// rosterStorage defines storage oprations for user's roster
type rosterStorage interface {
	InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error)
	DeleteRosterItem(username, jid string) (rostermodel.Version, error)
	FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error)
	FetchRosterItemsInGroups(username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error)
	FetchRosterItem(username, jid string) (*rostermodel.Item, error)
	InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error
	DeleteRosterNotification(contact, jid string) error
	FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error)
	FetchRosterNotifications(contact string) ([]rostermodel.Notification, error)
}

// InsertOrUpdateRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
	return instance().InsertOrUpdateRosterItem(ri)
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

// InsertOrUpdateRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error {
	return instance().InsertOrUpdateRosterNotification(rn)
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
