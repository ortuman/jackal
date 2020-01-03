/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model"
)

type badgerDBCapabilities struct {
	*badgerDBStorage
}

func newCapabilities(db *badger.DB) *badgerDBCapabilities {
	return &badgerDBCapabilities{badgerDBStorage: newStorage(db)}
}

func (b *badgerDBCapabilities) UpsertCapabilities(_ context.Context, caps *model.Capabilities) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(caps, capabilitiesKey(caps.Node, caps.Ver), tx)
	})
}

func (b *badgerDBCapabilities) FetchCapabilities(_ context.Context, node, ver string) (*model.Capabilities, error) {
	var caps model.Capabilities
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&caps, capabilitiesKey(node, ver), txn)
	})
	switch err {
	case nil:
		return &caps, nil
	case errEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func capabilitiesKey(node, ver string) []byte {
	return []byte("capabilities:" + node + ":" + ver)
}
