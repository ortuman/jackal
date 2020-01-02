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

// UpsertPrivateXML inserts a new private element into storage, or updates it in case it's been previously inserted.
func (b *Storage) UpsertPrivateXML(_ context.Context, privateXML []xmpp.XElement, namespace string, username string) error {
	r := xmpp.NewElementName("r")
	r.AppendElements(privateXML)
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(r, b.privateElementsKey(username, namespace), tx)
	})
}

// FetchPrivateXML retrieves from storage a private element.
func (b *Storage) FetchPrivateXML(_ context.Context, namespace string, username string) ([]xmpp.XElement, error) {
	var r xmpp.Element
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&r, b.privateElementsKey(username, namespace), txn)
	})
	switch err {
	case nil:
		return r.Elements().All(), nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *Storage) privateElementsKey(username, namespace string) []byte {
	return []byte("privateElements:" + username + ":" + namespace)
}
