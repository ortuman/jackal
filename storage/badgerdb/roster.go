/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model/rostermodel"
)

// InsertOrUpdateRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func (b *Storage) InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(ri, b.rosterItemKey(ri.Username, ri.JID), tx)
	}); err != nil {
		return rostermodel.Version{}, err
	}
	return b.updateRosterVer(ri.Username, false)
}

// DeleteRosterItem deletes a roster item entity from storage.
func (b *Storage) DeleteRosterItem(user, contact string) (rostermodel.Version, error) {
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.rosterItemKey(user, contact), tx)
	}); err != nil {
		return rostermodel.Version{}, err
	}
	return b.updateRosterVer(user, true)
}

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func (b *Storage) FetchRosterItems(user string) ([]rostermodel.Item, rostermodel.Version, error) {
	var ris []rostermodel.Item
	if err := b.fetchAll(&ris, []byte("rosterItems:"+user)); err != nil {
		return nil, rostermodel.Version{}, err
	}
	ver, err := b.fetchRosterVer(user)
	return ris, ver, err
}

// FetchRosterItemsInGroups retrieves from storage all roster item entities
// associated to a given user and a set of groups.
func (b *Storage) FetchRosterItemsInGroups(user string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupSet[group] = struct{}{}
	}
	// fetch all items
	var ris []rostermodel.Item
	if err := b.fetchAll(&ris, []byte("rosterItems:"+user)); err != nil {
		return nil, rostermodel.Version{}, err
	}
	var res []rostermodel.Item
	for _, ri := range ris {
		for _, riGroup := range ri.Groups {
			if _, ok := groupSet[riGroup]; ok {
				res = append(res, ri)
				break
			}
		}
	}
	ver, err := b.fetchRosterVer(user)
	return res, ver, err
}

// FetchRosterItem retrieves from storage a roster item entity.
func (b *Storage) FetchRosterItem(user, contact string) (*rostermodel.Item, error) {
	var ri rostermodel.Item
	err := b.fetch(&ri, b.rosterItemKey(user, contact))
	switch err {
	case nil:
		return &ri, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

// InsertOrUpdateRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func (b *Storage) InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(rn, b.rosterNotificationKey(rn.Contact, rn.JID), tx)
	})
}

// DeleteRosterNotification deletes a roster notification entity from storage.
func (b *Storage) DeleteRosterNotification(contact, jid string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.rosterNotificationKey(contact, jid), tx)
	})
}

// FetchRosterNotification retrieves from storage a roster notification entity.
func (b *Storage) FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error) {
	var rn rostermodel.Notification
	err := b.fetch(&rn, b.rosterNotificationKey(contact, jid))
	switch err {
	case nil:
		return &rn, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func (b *Storage) FetchRosterNotifications(contact string) ([]rostermodel.Notification, error) {
	var rns []rostermodel.Notification
	if err := b.fetchAll(&rns, []byte("rosterNotifications:"+contact)); err != nil {
		return nil, err
	}
	return rns, nil
}

func (b *Storage) updateRosterVer(username string, isDeletion bool) (rostermodel.Version, error) {
	v, err := b.fetchRosterVer(username)
	if err != nil {
		return rostermodel.Version{}, err
	}
	v.Ver++
	if isDeletion {
		v.DeletionVer = v.Ver
	}
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(&v, b.rosterVersionKey(username), tx)
	}); err != nil {
		return rostermodel.Version{}, err
	}
	return v, nil
}

func (b *Storage) fetchRosterVer(username string) (rostermodel.Version, error) {
	var ver rostermodel.Version
	err := b.fetch(&ver, b.rosterVersionKey(username))
	switch err {
	case nil, errBadgerDBEntityNotFound:
		return ver, nil
	default:
		return ver, err
	}
}

func (b *Storage) rosterItemKey(user, contact string) []byte {
	return []byte("rosterItems:" + user + ":" + contact)
}

func (b *Storage) rosterVersionKey(username string) []byte {
	return []byte("rosterVersions:" + username)
}

func (b *Storage) rosterNotificationKey(contact, jid string) []byte {
	return []byte("rosterNotifications:" + contact + ":" + jid)
}
