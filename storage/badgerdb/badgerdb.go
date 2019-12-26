/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model/serializer"
	"github.com/ortuman/jackal/pool"
)

var (
	errBadgerDBWrongEntityType = errors.New("badgerdb: wrong entity type")
	errBadgerDBEntityNotFound  = errors.New("badgerdb: entity not found")
)

// Storage represents a BadgerDB storage sub system.
type Storage struct {
	db     *badger.DB
	pool   *pool.BufferPool
	doneCh chan chan bool
}

// New returns a new BadgerDB storage instance.
func New(cfg *Config) *Storage {
	b := &Storage{
		pool:   pool.NewBufferPool(),
		doneCh: make(chan chan bool),
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DataDir), os.ModePerm); err != nil {
		log.Fatalf("%v", err)
	}
	opts := badger.DefaultOptions(cfg.DataDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("%v", err)
	}
	b.db = db
	go b.loop()
	return b
}

// IsClusterCompatible returns whether or not the underlying storage subsystem can be used in cluster mode.
func (b *Storage) IsClusterCompatible() bool { return false }

// Close shuts down BadgerDB storage sub system.
func (b *Storage) Close() error {
	ch := make(chan bool)
	b.doneCh <- ch
	<-ch
	return nil
}

func (b *Storage) loop() {
	tc := time.NewTicker(time.Minute)
	defer tc.Stop()
	for {
		select {
		case <-tc.C:
			if err := b.db.RunValueLogGC(0.5); err != nil {
				log.Warnf("%s", err)
			}
		case ch := <-b.doneCh:
			if err := b.db.Close(); err != nil {
				log.Warnf("%s", err)
			}
			close(ch)
			return
		}
	}
}

func (b *Storage) getVal(key []byte, txn *badger.Txn) ([]byte, error) {
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

func (b *Storage) setVal(key []byte, bts []byte, tx *badger.Txn) error {
	val := make([]byte, len(bts))
	copy(val, bts)
	return tx.Set(key, val)
}

func (b *Storage) fetch(entity interface{}, key []byte, txn *badger.Txn) error {
	val, err := b.getVal(key, txn)
	if err != nil {
		return err
	}
	if val != nil {
		if entity != nil {
			gd, ok := entity.(serializer.Deserializer)
			if !ok {
				return fmt.Errorf("%v: %T", errBadgerDBWrongEntityType, entity)
			}
			return serializer.Deserialize(val, gd)
		}
		return nil
	}
	return errBadgerDBEntityNotFound
}

func (b *Storage) upsert(entity interface{}, key []byte, tx *badger.Txn) error {
	gs, ok := entity.(serializer.Serializer)
	if !ok {
		return fmt.Errorf("%v: %T", errBadgerDBWrongEntityType, entity)
	}
	val, err := serializer.Serialize(gs)
	if err != nil {
		return err
	}
	return b.setVal(key, val, tx)
}

func (b *Storage) fetchSlice(slice interface{}, key []byte, tx *badger.Txn) error {
	val, err := b.getVal(key, tx)
	if err != nil {
		return err
	}
	if val == nil {
		return nil
	}
	return serializer.DeserializeSlice(val, slice)
}

func (b *Storage) upsertSlice(slice interface{}, key []byte, tx *badger.Txn) error {
	val, err := serializer.SerializeSlice(slice)
	if err != nil {
		return err
	}
	return b.setVal(key, val, tx)
}

func (b *Storage) delete(key []byte, txn *badger.Txn) error {
	return txn.Delete(key)
}

func (b *Storage) forEachKey(prefix []byte, f func(k []byte) error) error {
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
