/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/xml"
)

// InsertOrUpdateVCard inserts a new vCard element into storage,
// or updates it in case it's been previously inserted.
func (b *Storage) InsertOrUpdateVCard(vCard xml.XElement, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(vCard, b.vCardKey(username), tx)
	})
}

// FetchVCard retrieves from storage a vCard element associated
// to a given user.
func (b *Storage) FetchVCard(username string) (xml.XElement, error) {
	var vCard xml.Element
	err := b.fetch(&vCard, b.vCardKey(username))
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
