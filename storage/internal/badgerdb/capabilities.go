package badgerdb

import (
	"context"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model"
)

func (b *Storage) InsertCapabilities(_ context.Context, caps *model.Capabilities) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(caps, b.capabilitiesKey(caps.Node, caps.Ver), tx)
	})
}

func (b *Storage) FetchCapabilities(_ context.Context, node, ver string) (*model.Capabilities, error) {
	var caps model.Capabilities
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&caps, b.capabilitiesKey(node, ver), txn)
	})
	switch err {
	case nil:
		return &caps, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *Storage) capabilitiesKey(node, ver string) []byte {
	return []byte("capabilities:" + node + ":" + ver)
}
