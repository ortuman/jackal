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

type badgerDBPrivate struct {
	*badgerDBStorage
}

func newPrivate(db *badger.DB) *badgerDBPrivate {
	return &badgerDBPrivate{badgerDBStorage: newStorage(db)}
}

func (b *badgerDBPrivate) UpsertPrivateXML(_ context.Context, privateXML []xmpp.XElement, namespace string, username string) error {
	r := xmpp.NewElementName("r")
	r.AppendElements(privateXML)
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(r, privateElementsKey(username, namespace), tx)
	})
}

func (b *badgerDBPrivate) FetchPrivateXML(_ context.Context, namespace string, username string) ([]xmpp.XElement, error) {
	var r xmpp.Element
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&r, privateElementsKey(username, namespace), txn)
	})
	switch err {
	case nil:
		return r.Elements().All(), nil
	case errEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func privateElementsKey(username, namespace string) []byte {
	return []byte("privateElements:" + username + ":" + namespace)
}
