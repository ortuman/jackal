/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
)

type badgerDB struct {
	db     *badger.DB
	doneCh chan chan bool
}

func newBadgerDB(cfg *config.BadgerDb) *badgerDB {
	b := &badgerDB{doneCh: make(chan chan bool)}
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
	buf := pool.Get()
	defer pool.Put(buf)

	return b.db.Update(func(tx *badger.Txn) error {
		user.ToBytes(buf)
		return tx.Set(b.userKey(user.Username), buf.Bytes())
	})
}

func (b *badgerDB) DeleteUser(username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(b.userKey(username))
	})
}

func (b *badgerDB) FetchUser(username string) (*model.User, error) {
	var usr model.User
	if err := b.db.View(func(tx *badger.Txn) error {
		val, err := b.getVal(b.userKey(username), tx)
		if err != nil {
			return err
		}
		if val != nil {
			usr.FromBytes(bytes.NewReader(val))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &usr, nil
}

func (b *badgerDB) UserExists(username string) (bool, error) {
	var exists bool
	if err := b.db.View(func(tx *badger.Txn) error {
		val, err := b.getVal(b.userKey(username), tx)
		if err != nil {
			return err
		}
		exists = val != nil
		return nil
	}); err != nil {
		return false, err
	}
	return exists, nil
}

func (b *badgerDB) InsertOrUpdateRosterItem(ri *model.RosterItem) error {
	buf := pool.Get()
	defer pool.Put(buf)

	return b.db.Update(func(tx *badger.Txn) error {
		ri.ToBytes(buf)
		return tx.Set(b.rosterItemKey(ri.User, ri.Contact), buf.Bytes())
	})
}

func (b *badgerDB) DeleteRosterItem(user, contact string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(b.rosterItemKey(user, contact))
	})
}

func (b *badgerDB) FetchRosterItems(user string) ([]model.RosterItem, error) {
	var ris []model.RosterItem

	prefix := []byte("rosterItems:" + user)
	err := b.forEachKeyAndValue(prefix, func(k, val []byte) error {
		var ri model.RosterItem
		ri.FromBytes(bytes.NewReader(val))
		ris = append(ris, ri)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ris, nil
}

func (b *badgerDB) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	var ri model.RosterItem
	if err := b.db.View(func(tx *badger.Txn) error {
		val, err := b.getVal(b.rosterItemKey(user, contact), tx)
		if err != nil {
			return err
		}
		if val != nil {
			ri.FromBytes(bytes.NewReader(val))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &ri, nil
}

func (b *badgerDB) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	buf := pool.Get()
	defer pool.Put(buf)

	return b.db.Update(func(tx *badger.Txn) error {
		rn.ToBytes(buf)
		return tx.Set(b.rosterNotificationKey(rn.User, rn.Contact), buf.Bytes())
	})
}

func (b *badgerDB) DeleteRosterNotification(user, contact string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(b.rosterNotificationKey(user, contact))
	})
}

func (b *badgerDB) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	var rns []model.RosterNotification

	prefix := []byte("rosterNotifications:" + contact)
	err := b.forEachKeyAndValue(prefix, func(k, val []byte) error {
		var rn model.RosterNotification
		rn.FromBytes(bytes.NewReader(val))
		rns = append(rns, rn)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rns, nil
}

func (b *badgerDB) InsertOrUpdateVCard(vCard xml.Element, username string) error {
	buf := pool.Get()
	defer pool.Put(buf)

	return b.db.Update(func(tx *badger.Txn) error {
		vCard.ToBytes(buf)
		return tx.Set(b.vCardKey(username), buf.Bytes())
	})
}

func (b *badgerDB) FetchVCard(username string) (xml.Element, error) {
	var vCard xml.Element
	if err := b.db.View(func(tx *badger.Txn) error {
		val, err := b.getVal(b.vCardKey(username), tx)
		if err != nil {
			return err
		}
		if val != nil {
			var vc xml.MutableElement
			vc.FromBytes(bytes.NewReader(val))
			vCard = &vc
		}
		return err
	}); err != nil {
		return nil, err
	}
	return vCard, nil
}

func (b *badgerDB) InsertOrUpdatePrivateXML(privateXML []xml.Element, namespace string, username string) error {
	buf := pool.Get()
	defer pool.Put(buf)

	return b.db.Update(func(tx *badger.Txn) error {
		root := xml.NewElementName("r")
		root.AppendElements(privateXML)
		root.ToBytes(buf)
		return tx.Set(b.privateStorageKey(username, namespace), buf.Bytes())
	})
}

func (b *badgerDB) FetchPrivateXML(namespace string, username string) ([]xml.Element, error) {
	var privateXML []xml.Element
	if err := b.db.View(func(tx *badger.Txn) error {
		val, err := b.getVal(b.privateStorageKey(username, namespace), tx)
		if err != nil {
			return err
		}
		if val != nil {
			var root xml.MutableElement
			root.FromBytes(bytes.NewReader(val))
			privateXML = root.Elements()
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return privateXML, nil
}

func (b *badgerDB) InsertOfflineMessage(message xml.Element, username string) error {
	buf := pool.Get()
	defer pool.Put(buf)

	return b.db.Update(func(tx *badger.Txn) error {
		message.ToBytes(buf)
		return tx.Set(b.offlineMessageKey(username, message.ID()), buf.Bytes())
	})
}

func (b *badgerDB) CountOfflineMessages(username string) (int, error) {
	cnt := 0
	prefix := []byte("offlineMessages:" + username)
	err := b.forEachKey(prefix, func(key []byte) error {
		cnt++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (b *badgerDB) FetchOfflineMessages(username string) ([]xml.Element, error) {
	var msgs []xml.Element

	prefix := []byte("offlineMessages:" + username)
	err := b.forEachKeyAndValue(prefix, func(_, val []byte) error {
		var msg xml.MutableElement
		msg.FromBytes(bytes.NewReader(val))
		msgs = append(msgs, &msg)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (b *badgerDB) DeleteOfflineMessages(username string) error {
	var msgKeys [][]byte
	prefix := []byte("offlineMessages:" + username)
	err := b.forEachKey(prefix, func(key []byte) error {
		msgKeys = append(msgKeys, key)
		return nil
	})
	if err != nil {
		return err
	}
	return b.db.Update(func(txn *badger.Txn) error {
		for _, key := range msgKeys {
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
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

func (b *badgerDB) rosterNotificationKey(user, contact string) []byte {
	return []byte("rosterNotifications:" + contact + ":" + user)
}

func (b *badgerDB) offlineMessageKey(username, identifier string) []byte {
	return []byte("offlineMessages:" + username + ":" + identifier)
}

func (b *badgerDB) forEachKey(prefix []byte, f func(k []byte) error) error {
	return b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.AllVersions = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if err := f(it.Item().Key()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *badgerDB) forEachKeyAndValue(prefix []byte, f func(k, v []byte) error) error {
	return b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.AllVersions = false
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			val, err := it.Item().Value()
			if err != nil {
				return err
			}
			if err := f(it.Item().Key(), val); err != nil {
				return err
			}
		}
		return nil
	})
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
