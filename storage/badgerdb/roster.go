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

type badgerDBRoster struct {
	*badgerDBStorage
}

func newRoster(db *badger.DB) *badgerDBRoster {
	return &badgerDBRoster{badgerDBStorage: newStorage(db)}
}

func (b *badgerDBStorage) UpsertRosterItem(_ context.Context, rItem *rostermodel.Item) (rostermodel.Version, error) {
	var ris []rostermodel.Item
	var ver rostermodel.Version

	err := b.db.Update(func(tx *badger.Txn) error {
		if err := b.fetchSlice(&ris, rosterItemsKey(rItem.Username), tx); err != nil {
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
		if err := b.upsertSlice(&ris, rosterItemsKey(rItem.Username), tx); err != nil {
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

func (b *badgerDBStorage) DeleteRosterItem(_ context.Context, user, contact string) (rostermodel.Version, error) {
	var ver rostermodel.Version

	err := b.db.Update(func(tx *badger.Txn) error {
		var ris []rostermodel.Item
		if err := b.fetchSlice(&ris, rosterItemsKey(user), tx); err != nil {
			return err
		}
		for i, ri := range ris {
			if ri.JID == contact { // delete roster item
				ris = append(ris[:i], ris[i+1:]...)
				if err := b.upsertSlice(&ris, rosterItemsKey(user), tx); err != nil {
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

func (b *badgerDBStorage) FetchRosterItems(_ context.Context, user string) ([]rostermodel.Item, rostermodel.Version, error) {
	var ris []rostermodel.Item
	var ver rostermodel.Version

	err := b.db.View(func(txn *badger.Txn) error {
		if err := b.fetchSlice(&ris, rosterItemsKey(user), txn); err != nil {
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

func (b *badgerDBStorage) FetchRosterItemsInGroups(_ context.Context, user string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupSet[group] = struct{}{}
	}
	// fetch all items
	var ris []rostermodel.Item
	var ver rostermodel.Version

	err := b.db.View(func(txn *badger.Txn) error {
		if err := b.fetchSlice(&ris, rosterItemsKey(user), txn); err != nil {
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

func (b *badgerDBStorage) FetchRosterItem(_ context.Context, user, contact string) (*rostermodel.Item, error) {
	var ris []rostermodel.Item
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&ris, rosterItemsKey(user), txn)
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

func (b *badgerDBStorage) UpsertRosterNotification(_ context.Context, rNotification *rostermodel.Notification) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var rns []rostermodel.Notification
		if err := b.fetchSlice(&rns, rosterNotificationsKey(rNotification.Contact), tx); err != nil {
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
		return b.upsertSlice(&rns, rosterNotificationsKey(rNotification.Contact), tx)
	})
}

func (b *badgerDBStorage) DeleteRosterNotification(_ context.Context, contact, jid string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var rns []rostermodel.Notification
		if err := b.fetchSlice(&rns, rosterNotificationsKey(contact), tx); err != nil {
			return err
		}
		for i, rn := range rns {
			if rn.JID == jid {
				rns = append(rns[:i], rns[i+1:]...)
				return b.upsertSlice(&rns, rosterNotificationsKey(contact), tx)
			}
		}
		return nil
	})
}

func (b *badgerDBStorage) FetchRosterNotification(_ context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	var rns []rostermodel.Notification
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&rns, rosterNotificationsKey(contact), txn)
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

func (b *badgerDBStorage) FetchRosterNotifications(_ context.Context, contact string) ([]rostermodel.Notification, error) {
	var rns []rostermodel.Notification
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&rns, rosterNotificationsKey(contact), txn)
	})
	if err != nil {
		return nil, err
	}
	return rns, nil
}

func (b *badgerDBStorage) FetchRosterGroups(_ context.Context, username string) ([]string, error) {
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

func (b *badgerDBStorage) upsertRosterVer(username string, isDeletion bool, txn *badger.Txn) (rostermodel.Version, error) {
	v, err := b.fetchRosterVer(username, txn)
	if err != nil {
		return rostermodel.Version{}, err
	}
	v.Ver++
	if isDeletion {
		v.DeletionVer = v.Ver
	}
	if err := b.upsert(&v, rosterVersionsKey(username), txn); err != nil {
		return rostermodel.Version{}, err
	}
	return v, nil
}

func (b *badgerDBStorage) fetchRosterVer(username string, txn *badger.Txn) (rostermodel.Version, error) {
	var ver rostermodel.Version
	err := b.fetch(&ver, rosterVersionsKey(username), txn)
	switch err {
	case nil, errEntityNotFound:
		return ver, nil
	default:
		return ver, err
	}
}

func (b *badgerDBStorage) upsertRosterGroups(user string, ris []rostermodel.Item, tx *badger.Txn) error {
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
	return b.setVal(rosterGroupsKey(user), buf.Bytes(), tx)
}

func (b *badgerDBStorage) fetchRosterGroups(user string, txn *badger.Txn) ([]string, error) {
	var ln int
	var groups []string

	val, err := b.getVal(rosterGroupsKey(user), txn)
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

func rosterItemsKey(user string) []byte {
	return []byte("rosterItems:" + user)
}

func rosterNotificationsKey(contact string) []byte {
	return []byte("rosterNotifications:" + contact)
}

func rosterVersionsKey(username string) []byte {
	return []byte("rosterVersions:" + username)
}

func rosterGroupsKey(username string) []byte {
	return []byte("rosterGroups:" + username)
}
