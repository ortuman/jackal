/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model/serializer"
)

var (
	// errEntityNotFound represents an entity not found error
	errEntityNotFound = errors.New("badgerdb: entity not found")

	// errWrongEntityType represents an invalid entity type error
	errWrongEntityType = errors.New("badgerdb: wrong entity type")
)

// badgerDBStorage represents a BadgerDB base storage sub system.
type badgerDBStorage struct {
	db *badger.DB
}

// newStorage returns a new BadgerDB base storage instance.
func newStorage(db *badger.DB) *badgerDBStorage {
	return &badgerDBStorage{db: db}
}

// getVal looks for key and returns corresponding entity.
func (b *badgerDBStorage) getVal(key []byte, txn *badger.Txn) ([]byte, error) {
	item, err := txn.Get(key)
	switch err {
	case nil:
		break
	case badger.ErrKeyNotFound:
		return nil, nil
	default:
		return nil, err
	}
	return item.ValueCopy(nil)
}

// setVal adds a key-value pair to the database.
func (b *badgerDBStorage) setVal(key []byte, bts []byte, tx *badger.Txn) error {
	val := make([]byte, len(bts))
	copy(val, bts)
	return tx.Set(key, val)
}

// fetch retrieves and deserializes a database entity.
func (b *badgerDBStorage) fetch(entity interface{}, key []byte, txn *badger.Txn) error {
	val, err := b.getVal(key, txn)
	if err != nil {
		return err
	}
	if val != nil {
		if entity != nil {
			gd, ok := entity.(serializer.Deserializer)
			if !ok {
				return fmt.Errorf("%v: %T", errWrongEntityType, entity)
			}
			return serializer.Deserialize(val, gd)
		}
		return nil
	}
	return errEntityNotFound
}

// Upsert inserts or updates a serializable entity into database.
func (b *badgerDBStorage) upsert(entity interface{}, key []byte, tx *badger.Txn) error {
	gs, ok := entity.(serializer.Serializer)
	if !ok {
		return fmt.Errorf("%v: %T", errWrongEntityType, entity)
	}
	val, err := serializer.Serialize(gs)
	if err != nil {
		return err
	}
	return b.setVal(key, val, tx)
}

// fetchSlice inserts or updates a serializable entity into database.
func (b *badgerDBStorage) fetchSlice(slice interface{}, key []byte, tx *badger.Txn) error {
	val, err := b.getVal(key, tx)
	if err != nil {
		return err
	}
	if val == nil {
		return nil
	}
	return serializer.DeserializeSlice(val, slice)
}

// upsertSlice retrieves and deserializes a database slice.
func (b *badgerDBStorage) upsertSlice(slice interface{}, key []byte, tx *badger.Txn) error {
	val, err := serializer.SerializeSlice(slice)
	if err != nil {
		return err
	}
	return b.setVal(key, val, tx)
}

// delete deletes a key.
func (b *badgerDBStorage) delete(key []byte, txn *badger.Txn) error {
	return txn.Delete(key)
}

// forEachKey iterates all entities matching a given prefix.
func (b *badgerDBStorage) forEachKey(prefix []byte, f func(k []byte) error) error {
	return b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		iter := txn.NewIterator(opts)
		defer iter.Close()

		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			it := iter.Item()
			if err := f(it.Key()); err != nil {
				return err
			}
		}
		return nil
	})
}
