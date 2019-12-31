/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/dgraph-io/badger"
	rostermodel "github.com/ortuman/jackal/model/roster"
)

// UpsertRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func (b *Storage) UpsertRosterItem(_ context.Context, rItem *rostermodel.Item) (rostermodel.Version, error) {
	var ris []rostermodel.Item
	var ver rostermodel.Version

	err := b.db.Update(func(tx *badger.Txn) error {
		if err := b.fetchSlice(&ris, b.rosterItemsKey(rItem.Username), tx); err != nil {
			return err
		}
		var updated bool
		for i, ri := range ris {
			if ri.JID == rItem.JID {
				ris[i] = *rItem
				updated = true
				break
			}
		}
		if !updated {
			ris = append(ris, *rItem)
		}
		if err := b.upsertSlice(&ris, b.rosterItemsKey(rItem.Username), tx); err != nil {
			return err
		}
		// update roster groups
		if err := b.upsertRosterGroups(rItem.Username, ris, tx); err != nil {
			return err
		}
		// update roster version
		v, err := b.upsertRosterVer(rItem.Username, false, tx)
		if err != nil {
			return err
		}
		ver = v
		return nil
	})
	if err != nil {
		return rostermodel.Version{}, err
	}
	return ver, nil
}

// DeleteRosterItem deletes a roster item entity from storage.
func (b *Storage) DeleteRosterItem(_ context.Context, user, contact string) (rostermodel.Version, error) {
	var ver rostermodel.Version

	err := b.db.Update(func(tx *badger.Txn) error {
		var ris []rostermodel.Item
		if err := b.fetchSlice(&ris, b.rosterItemsKey(user), tx); err != nil {
			return err
		}
		for i, ri := range ris {
			if ri.JID == contact { // delete roster item
				ris = append(ris[:i], ris[i+1:]...)
				if err := b.upsertSlice(&ris, b.rosterItemsKey(user), tx); err != nil {
					return err
				}
				break
			}
		}
		// update roster groups
		if err := b.upsertRosterGroups(user, ris, tx); err != nil {
			return err
		}
		// update roster version
		v, err := b.upsertRosterVer(user, true, tx)
		if err != nil {
			return err
		}
		ver = v
		return nil
	})
	if err != nil {
		return rostermodel.Version{}, err
	}
	return ver, nil
}

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func (b *Storage) FetchRosterItems(_ context.Context, user string) ([]rostermodel.Item, rostermodel.Version, error) {
	var ris []rostermodel.Item
	var ver rostermodel.Version

	err := b.db.View(func(txn *badger.Txn) error {
		if err := b.fetchSlice(&ris, b.rosterItemsKey(user), txn); err != nil {
			return err
		}
		v, err := b.fetchRosterVer(user, txn)
		if err != nil {
			return err
		}
		ver = v
		return nil
	})
	if err != nil {
		return nil, rostermodel.Version{}, err
	}
	return ris, ver, err
}

// FetchRosterItemsInGroups retrieves from storage all roster item entities
// associated to a given user and a set of groups.
func (b *Storage) FetchRosterItemsInGroups(_ context.Context, user string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupSet[group] = struct{}{}
	}
	// fetch all items
	var ris []rostermodel.Item
	var ver rostermodel.Version

	err := b.db.View(func(txn *badger.Txn) error {
		if err := b.fetchSlice(&ris, b.rosterItemsKey(user), txn); err != nil {
			return err
		}
		v, err := b.fetchRosterVer(user, txn)
		if err != nil {
			return err
		}
		ver = v
		return nil
	})
	if err != nil {
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
	return res, ver, err
}

// FetchRosterItem retrieves from storage a roster item entity.
func (b *Storage) FetchRosterItem(_ context.Context, user, contact string) (*rostermodel.Item, error) {
	var ris []rostermodel.Item
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&ris, b.rosterItemsKey(user), txn)
	})
	if err != nil {
		return nil, err
	}
	for _, ri := range ris {
		if ri.JID == contact {
			return &ri, nil
		}
	}
	return nil, nil
}

// UpsertRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func (b *Storage) UpsertRosterNotification(_ context.Context, rNotification *rostermodel.Notification) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var rns []rostermodel.Notification
		if err := b.fetchSlice(&rns, b.rosterNotificationsKey(rNotification.Contact), tx); err != nil {
			return err
		}
		var updated bool
		for i, rn := range rns {
			if rn.JID == rNotification.JID {
				rns[i] = *rNotification
				updated = true
				break
			}
		}
		if !updated {
			rns = append(rns, *rNotification)
		}
		return b.upsertSlice(&rns, b.rosterNotificationsKey(rNotification.Contact), tx)
	})
}

// DeleteRosterNotification deletes a roster notification entity from storage.
func (b *Storage) DeleteRosterNotification(_ context.Context, contact, jid string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var rns []rostermodel.Notification
		if err := b.fetchSlice(&rns, b.rosterNotificationsKey(contact), tx); err != nil {
			return err
		}
		for i, rn := range rns {
			if rn.JID == jid {
				rns = append(rns[:i], rns[i+1:]...)
				return b.upsertSlice(&rns, b.rosterNotificationsKey(contact), tx)
			}
		}
		return nil
	})
}

// FetchRosterNotification retrieves from storage a roster notification entity.
func (b *Storage) FetchRosterNotification(_ context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	var rns []rostermodel.Notification
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&rns, b.rosterNotificationsKey(contact), txn)
	})
	if err != nil {
		return nil, err
	}
	for _, rn := range rns {
		if rn.JID == jid {
			return &rn, nil
		}
	}
	return nil, nil
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func (b *Storage) FetchRosterNotifications(_ context.Context, contact string) ([]rostermodel.Notification, error) {
	var rns []rostermodel.Notification
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&rns, b.rosterNotificationsKey(contact), txn)
	})
	if err != nil {
		return nil, err
	}
	return rns, nil
}

// FetchRosterGroups retrieves all groups associated to a user roster
func (b *Storage) FetchRosterGroups(_ context.Context, username string) ([]string, error) {
	var groups []string
	err := b.db.View(func(txn *badger.Txn) error {
		var fnErr error
		groups, fnErr = b.fetchRosterGroups(username, txn)
		return fnErr
	})
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (b *Storage) upsertRosterVer(username string, isDeletion bool, txn *badger.Txn) (rostermodel.Version, error) {
	v, err := b.fetchRosterVer(username, txn)
	if err != nil {
		return rostermodel.Version{}, err
	}
	v.Ver++
	if isDeletion {
		v.DeletionVer = v.Ver
	}
	if err := b.upsert(&v, b.rosterVersionsKey(username), txn); err != nil {
		return rostermodel.Version{}, err
	}
	return v, nil
}

func (b *Storage) fetchRosterVer(username string, txn *badger.Txn) (rostermodel.Version, error) {
	var ver rostermodel.Version
	err := b.fetch(&ver, b.rosterVersionsKey(username), txn)
	switch err {
	case nil, errBadgerDBEntityNotFound:
		return ver, nil
	default:
		return ver, err
	}
}

func (b *Storage) upsertRosterGroups(user string, ris []rostermodel.Item, tx *badger.Txn) error {
	var groupsSet = make(map[string]struct{})
	// remove duplicates
	for _, ri := range ris {
		for _, group := range ri.Groups {
			groupsSet[group] = struct{}{}
		}
	}
	var groups []string
	for group := range groupsSet {
		groups = append(groups, group)
	}
	// encode groups
	buf := bytes.NewBuffer(nil)

	enc := gob.NewEncoder(buf)
	if err := enc.Encode(len(groups)); err != nil {
		return err
	}
	for _, group := range groups {
		if err := enc.Encode(group); err != nil {
			return err
		}
	}
	return b.setVal(b.rosterGroupsKey(user), buf.Bytes(), tx)
}

func (b *Storage) fetchRosterGroups(user string, txn *badger.Txn) ([]string, error) {
	var ln int
	var groups []string

	val, err := b.getVal(b.rosterGroupsKey(user), txn)
	if err != nil {
		return nil, err
	}
	// decode groups
	dec := gob.NewDecoder(bytes.NewReader(val))
	if err := dec.Decode(&ln); err != nil {
		return nil, err
	}
	for i := 0; i < ln; i++ {
		var group string
		if err := dec.Decode(&group); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (b *Storage) rosterItemsKey(user string) []byte {
	return []byte("rosterItems:" + user)
}

func (b *Storage) rosterNotificationsKey(contact string) []byte {
	return []byte("rosterNotifications:" + contact)
}

func (b *Storage) rosterVersionsKey(username string) []byte {
	return []byte("rosterVersions:" + username)
}

func (b *Storage) rosterGroupsKey(username string) []byte {
	return []byte("rosterGroups:" + username)
}
