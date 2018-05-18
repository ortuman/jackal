/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/storage/model"
)

func (b *Storage) InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error) {
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(ri, b.rosterItemKey(ri.Username, ri.JID), tx)
	}); err != nil {
		return model.RosterVersion{}, err
	}
	return b.updateRosterVer(ri.Username, false)
}

func (b *Storage) DeleteRosterItem(user, contact string) (model.RosterVersion, error) {
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.rosterItemKey(user, contact), tx)
	}); err != nil {
		return model.RosterVersion{}, err
	}
	return b.updateRosterVer(user, true)
}

func (b *Storage) FetchRosterItems(user string) ([]model.RosterItem, model.RosterVersion, error) {
	var ris []model.RosterItem
	if err := b.fetchAll(&ris, []byte("rosterItems:"+user)); err != nil {
		return nil, model.RosterVersion{}, err
	}
	ver, err := b.fetchRosterVer(user)
	return ris, ver, err
}

func (b *Storage) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	var ri model.RosterItem
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

func (b *Storage) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(rn, b.rosterNotificationKey(rn.Contact, rn.JID), tx)
	})
}

func (b *Storage) DeleteRosterNotification(contact, jid string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.rosterNotificationKey(contact, jid), tx)
	})
}

func (b *Storage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	var rns []model.RosterNotification
	if err := b.fetchAll(&rns, []byte("rosterNotifications:"+contact)); err != nil {
		return nil, err
	}
	return rns, nil
}

func (b *Storage) updateRosterVer(username string, isDeletion bool) (model.RosterVersion, error) {
	v, err := b.fetchRosterVer(username)
	if err != nil {
		return model.RosterVersion{}, err
	}
	v.Ver++
	if isDeletion {
		v.DeletionVer = v.Ver
	}
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(&v, b.rosterVersionKey(username), tx)
	}); err != nil {
		return model.RosterVersion{}, err
	}
	return v, nil
}

func (b *Storage) fetchRosterVer(username string) (model.RosterVersion, error) {
	var ver model.RosterVersion
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
