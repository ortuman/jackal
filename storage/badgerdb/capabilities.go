package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model"
)

func (b *Storage) InsertCapabilities(node, ver string, caps *model.Capabilities) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(caps, b.capabilitiesKey(node, ver), tx)
	})
}

func (b *Storage) HasCapabilities(node, ver string) (bool, error) {
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(nil, b.capabilitiesKey(node, ver), txn)
	})
	switch err {
	case nil:
		return true, nil
	case errBadgerDBEntityNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (b *Storage) FetchCapabilities(node, ver string) (*model.Capabilities, error) {
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
