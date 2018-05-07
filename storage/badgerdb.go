/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
)

var (
	errBadgerDBWrongEntityType = errors.New("badgerdb: wrong entity type")
	errBadgerDBEntityNotFound  = errors.New("badgerdb: entity not found")
)

type badgerDB struct {
	db     *badger.DB
	pool   *pool.BufferPool
	doneCh chan chan bool
}

func newBadgerDB(cfg *config.BadgerDb) *badgerDB {
	b := &badgerDB{
		pool:   pool.NewBufferPool(),
		doneCh: make(chan chan bool),
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DataDir), os.ModePerm); err != nil {
		log.Fatalf("%v", err)
	}
	opts := badger.DefaultOptions
	opts.Dir = cfg.DataDir
	opts.ValueDir = cfg.DataDir
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("%v", err)
	}
	b.db = db
	go b.loop()
	return b
}

func (b *badgerDB) Shutdown() {
	ch := make(chan bool)
	b.doneCh <- ch
	<-ch
}

func (b *badgerDB) InsertOrUpdateUser(user *model.User) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(user, b.userKey(user.Username), tx)
	})
}

func (b *badgerDB) DeleteUser(username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.userKey(username), tx)
	})
}

func (b *badgerDB) FetchUser(username string) (*model.User, error) {
	var usr model.User
	err := b.fetch(&usr, b.userKey(username))
	switch err {
	case nil:
		return &usr, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *badgerDB) UserExists(username string) (bool, error) {
	err := b.fetch(nil, b.userKey(username))
	switch err {
	case nil:
		return true, nil
	case errBadgerDBEntityNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (b *badgerDB) InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error) {
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(ri, b.rosterItemKey(ri.Username, ri.JID), tx)
	}); err != nil {
		return model.RosterVersion{}, err
	}
	return b.updateRosterVer(ri.Username, false)
}

func (b *badgerDB) DeleteRosterItem(user, contact string) (model.RosterVersion, error) {
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.rosterItemKey(user, contact), tx)
	}); err != nil {
		return model.RosterVersion{}, err
	}
	return b.updateRosterVer(user, true)
}

func (b *badgerDB) FetchRosterItems(user string) ([]model.RosterItem, model.RosterVersion, error) {
	var ris []model.RosterItem
	if err := b.fetchAll(&ris, []byte("rosterItems:"+user)); err != nil {
		return nil, model.RosterVersion{}, err
	}
	ver, err := b.fetchRosterVer(user)
	return ris, ver, err
}

func (b *badgerDB) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	var ri model.RosterItem
	err := b.fetch(&ri, b.rosterItemKey(user, contact))
	switch err {
	case nil:
		return &ri, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *badgerDB) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(rn, b.rosterNotificationKey(rn.Contact, rn.JID), tx)
	})
}

func (b *badgerDB) DeleteRosterNotification(contact, jid string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.rosterNotificationKey(contact, jid), tx)
	})
}

func (b *badgerDB) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	var rns []model.RosterNotification
	if err := b.fetchAll(&rns, []byte("rosterNotifications:"+contact)); err != nil {
		return nil, err
	}
	return rns, nil
}

func (b *badgerDB) InsertOrUpdateVCard(vCard xml.XElement, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(vCard, b.vCardKey(username), tx)
	})
}

func (b *badgerDB) FetchVCard(username string) (xml.XElement, error) {
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

func (b *badgerDB) InsertOrUpdatePrivateXML(privateXML []xml.XElement, namespace string, username string) error {
	r := xml.NewElementName("r")
	r.AppendElements(privateXML)
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(r, b.privateStorageKey(username, namespace), tx)
	})
}

func (b *badgerDB) FetchPrivateXML(namespace string, username string) ([]xml.XElement, error) {
	var r xml.Element
	err := b.fetch(&r, b.privateStorageKey(username, namespace))
	switch err {
	case nil:
		return r.Elements().All(), nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *badgerDB) InsertOfflineMessage(message xml.XElement, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(message, b.offlineMessageKey(username, message.ID()), tx)
	})
}

func (b *badgerDB) CountOfflineMessages(username string) (int, error) {
	cnt := 0
	prefix := []byte("offlineMessages:" + username)
	err := b.forEachKey(prefix, func(key []byte) error {
		cnt++
		return nil
	})
	return cnt, err
}

func (b *badgerDB) FetchOfflineMessages(username string) ([]xml.XElement, error) {
	var msgs []xml.Element
	if err := b.fetchAll(&msgs, []byte("offlineMessages:"+username)); err != nil {
		return nil, err
	}
	switch len(msgs) {
	case 0:
		return nil, nil
	default:
		ret := make([]xml.XElement, len(msgs))
		for i := 0; i < len(msgs); i++ {
			ret[i] = &msgs[i]
		}
		return ret, nil
	}
}

func (b *badgerDB) DeleteOfflineMessages(username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.deletePrefix([]byte("offlineMessages:"+username), tx)
	})
}

func (b *badgerDB) InsertOrUpdateBlockListItems(items []model.BlockListItem) error {
	return b.db.Update(func(tx *badger.Txn) error {
		for _, item := range items {
			if err := b.insertOrUpdate(&item, b.blockListItemKey(item.Username, item.JID), tx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *badgerDB) DeleteBlockListItems(items []model.BlockListItem) error {
	return b.db.Update(func(tx *badger.Txn) error {
		for _, item := range items {
			if err := b.delete(b.blockListItemKey(item.Username, item.JID), tx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *badgerDB) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	var blItems []model.BlockListItem
	if err := b.fetchAll(&blItems, []byte("blockListItems:"+username)); err != nil {
		return nil, err
	}
	return blItems, nil
}

func (b *badgerDB) updateRosterVer(username string, isDeletion bool) (model.RosterVersion, error) {
	v, err := b.fetchRosterVer(username)
	if err != nil {
		return model.RosterVersion{}, err
	}
	v.Ver++
	if isDeletion {
		v.DeletionVer = v.Ver
	}
	if err := b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(&v, b.rosterVersionKey(username), tx)
	}); err != nil {
		return model.RosterVersion{}, err
	}
	return v, nil
}

func (b *badgerDB) fetchRosterVer(username string) (model.RosterVersion, error) {
	var ver model.RosterVersion
	err := b.fetch(&ver, b.rosterVersionKey(username))
	switch err {
	case nil, errBadgerDBEntityNotFound:
		return ver, nil
	default:
		return ver, err
	}
}

func (b *badgerDB) loop() {
	tc := time.NewTicker(time.Minute)
	defer tc.Stop()
	for {
		select {
		case <-tc.C:
			b.db.PurgeOlderVersions()
			b.db.RunValueLogGC(0.5)
		case ch := <-b.doneCh:
			b.db.Close()
			close(ch)
			return
		}
	}
}

func (b *badgerDB) insertOrUpdate(entity interface{}, key []byte, tx *badger.Txn) error {
	gs, ok := entity.(model.GobSerializer)
	if !ok {
		return fmt.Errorf("%v: %T", errBadgerDBWrongEntityType, entity)
	}
	buf := b.pool.Get()
	defer b.pool.Put(buf)

	gs.ToGob(gob.NewEncoder(buf))
	bts := buf.Bytes()
	val := make([]byte, len(bts))
	copy(val, bts)
	return tx.Set(key, val)
}

func (b *badgerDB) delete(key []byte, txn *badger.Txn) error {
	return txn.Delete(key)
}

func (b *badgerDB) deletePrefix(prefix []byte, txn *badger.Txn) error {
	var keys [][]byte
	if err := b.forEachKey(prefix, func(key []byte) error {
		keys = append(keys, key)
		return nil
	}); err != nil {
		return err
	}
	for _, k := range keys {
		if err := txn.Delete(k); err != nil {
			return err
		}
	}
	return nil
}

func (b *badgerDB) fetch(entity interface{}, key []byte) error {
	return b.db.View(func(tx *badger.Txn) error {
		val, err := b.getVal(key, tx)
		if err != nil {
			return err
		}
		if val != nil {
			if entity != nil {
				gd, ok := entity.(model.GobDeserializer)
				if !ok {
					return fmt.Errorf("%v: %T", errBadgerDBWrongEntityType, entity)
				}
				gd.FromGob(gob.NewDecoder(bytes.NewReader(val)))
			}
			return nil
		}
		return errBadgerDBEntityNotFound
	})
}

func (b *badgerDB) fetchAll(v interface{}, prefix []byte) error {
	t := reflect.TypeOf(v).Elem()
	if t.Kind() != reflect.Slice {
		return fmt.Errorf("%v: %T", errBadgerDBWrongEntityType, v)
	}
	s := reflect.ValueOf(v).Elem()
	return b.forEachKeyAndValue(prefix, func(k, val []byte) error {
		e := reflect.New(t.Elem()).Elem()
		i := e.Addr().Interface()
		gd, ok := i.(model.GobDeserializer)
		if !ok {
			return fmt.Errorf("%v: %T", errBadgerDBWrongEntityType, i)
		}
		gd.FromGob(gob.NewDecoder(bytes.NewReader(val)))
		s.Set(reflect.Append(s, e))
		return nil
	})
	return nil
}

func (b *badgerDB) getVal(key []byte, txn *badger.Txn) ([]byte, error) {
	item, err := txn.Get(key)
	switch err {
	case nil:
		break
	case badger.ErrKeyNotFound:
		return nil, nil
	default:
		return nil, err
	}
	return item.Value()
}

func (b *badgerDB) forEachKey(prefix []byte, f func(k []byte) error) error {
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

func (b *badgerDB) forEachKeyAndValue(prefix []byte, f func(k, v []byte) error) error {
	return b.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			it := iter.Item()
			val, err := it.Value()
			if err != nil {
				return err
			}
			if err := f(it.Key(), val); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *badgerDB) userKey(username string) []byte {
	return []byte("users:" + username)
}

func (b *badgerDB) vCardKey(username string) []byte {
	return []byte("vCards:" + username)
}

func (b *badgerDB) privateStorageKey(username, namespace string) []byte {
	return []byte("privateElements:" + username + ":" + namespace)
}

func (b *badgerDB) rosterItemKey(user, contact string) []byte {
	return []byte("rosterItems:" + user + ":" + contact)
}

func (b *badgerDB) rosterVersionKey(username string) []byte {
	return []byte("rosterVersions:" + username)
}

func (b *badgerDB) rosterNotificationKey(contact, jid string) []byte {
	return []byte("rosterNotifications:" + contact + ":" + jid)
}

func (b *badgerDB) offlineMessageKey(username, identifier string) []byte {
	return []byte("offlineMessages:" + username + ":" + identifier)
}

func (b *badgerDB) blockListItemKey(username, jid string) []byte {
	return []byte("blockListItems:" + username + ":" + jid)
}
