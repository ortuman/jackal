/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/xmpp"
)

// UpsertVCard inserts a new vCard element into storage,
// or updates it in case it's been previously inserted.
func (b *Storage) UpsertVCard(vCard xmpp.XElement, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(vCard, b.vCardKey(username), tx)
	})
}

// FetchVCard retrieves from storage a vCard element associated
// to a given user.
func (b *Storage) FetchVCard(username string) (xmpp.XElement, error) {
	var vCard xmpp.Element
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&vCard, b.vCardKey(username), txn)
	})
	switch err {
	case nil:
		return &vCard, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *Storage) vCardKey(username string) []byte {
	return []byte("vCards:" + username)
}
