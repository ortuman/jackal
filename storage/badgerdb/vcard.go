/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/xmpp"
)

type badgerDBVCard struct {
	*badgerDBStorage
}

func newVCard(db *badger.DB) *badgerDBVCard {
	return &badgerDBVCard{badgerDBStorage: newStorage(db)}
}

// UpsertVCard inserts a new vCard element into storage, or updates it in case it's been previously inserted.
func (b *badgerDBVCard) UpsertVCard(_ context.Context, vCard xmpp.XElement, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(vCard, vCardKey(username), tx)
	})
}

// FetchVCard retrieves from storage a vCard element associated to a given user.
func (b *badgerDBVCard) FetchVCard(_ context.Context, username string) (xmpp.XElement, error) {
	var vCard xmpp.Element
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&vCard, vCardKey(username), txn)
	})
	switch err {
	case nil:
		return &vCard, nil
	case errEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func vCardKey(username string) []byte {
	return []byte("vCards:" + username)
}
