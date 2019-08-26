/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"bytes"
	"encoding/gob"

	"github.com/dgraph-io/badger"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

type itemSet struct {
	items []pubsubmodel.Item
}

func (s itemSet) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(len(s.items)); err != nil {
		return err
	}
	for _, itm := range s.items {
		if err := itm.ToBytes(buf); err != nil {
			return err
		}
	}
	return nil
}

func (s itemSet) FromBytes(buf *bytes.Buffer) error {
	var ln int
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&ln); err != nil {
		return err
	}
	for i := 0; i < ln; i++ {
		var itm pubsubmodel.Item
		if err := itm.FromBytes(buf); err != nil {
			return err
		}
		s.items = append(s.items, itm)
	}
	return nil
}

func (b *Storage) UpsertPubSubNode(node *pubsubmodel.Node) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(node, b.pubSubNodeStorageKey(node.Host, node.Name), tx)
	})
}

func (b *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	var node pubsubmodel.Node
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&node, b.pubSubNodeStorageKey(host, name), txn)
	})
	switch err {
	case nil:
		return &node, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *Storage) UpsertPubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var s itemSet
		if err := b.fetch(&s, b.pubSubItemsStorageKey(host, name), tx); err != nil {
			return err
		}
		var updated bool
		for i, itm := range s.items {
			if itm.ID == item.ID {
				s.items[i] = *item
				updated = true
				break
			}
		}
		if !updated {
			s.items = append(s.items, *item)
		}
		if len(s.items) > maxNodeItems {
			s.items = s.items[1:] // remove oldest element
		}
		return b.upsert(s, b.pubSubItemsStorageKey(host, name), tx)
	})
}

func (b *Storage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	var items []pubsubmodel.Item
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchAll(&items, []byte("pubSubItems:"+host+":"+name), txn)
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (b *Storage) UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(affiliation, b.pubSubAffiliationStorageKey(host, name, affiliation.JID), tx)
	})
}

func (b *Storage) FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	var affiliations []pubsubmodel.Affiliation
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchAll(&affiliations, []byte("pubSubAffiliations:"+host+":"+name), txn)
	})
	if err != nil {
		return nil, err
	}
	return affiliations, nil
}

func (b *Storage) pubSubNodeStorageKey(host, name string) []byte {
	return []byte("pubSubNodes:" + host + ":" + name)
}

func (b *Storage) pubSubItemsStorageKey(host, name string) []byte {
	return []byte("pubSubItems:" + host + ":" + name)
}

func (b *Storage) pubSubAffiliationStorageKey(host, name, jid string) []byte {
	return []byte("pubSubAffiliations:" + host + ":" + name + ":" + jid)
}
