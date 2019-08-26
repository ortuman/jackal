/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/model/serializer"
)

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
		val, err := b.getVal(b.pubSubItemsStorageKey(host, name), tx)
		if err != nil {
			return err
		}
		var items []pubsubmodel.Item
		if err := serializer.DeserializeSlice(val, &items); err != nil {
			return err
		}
		var updated bool
		for i, itm := range items {
			if itm.ID == item.ID {
				items[i] = *item
				updated = true
				break
			}
		}
		if !updated {
			items = append(items, *item)
		}
		if len(items) > maxNodeItems {
			items = items[1:] // remove oldest element
		}
		bts, err := serializer.SerializeSlice(&items)
		if err != nil {
			return err
		}
		return b.setVal(b.pubSubItemsStorageKey(host, name), bts, tx)
	})
}

func (b *Storage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	var items []pubsubmodel.Item
	err := b.db.View(func(txn *badger.Txn) error {
		val, err := b.getVal(b.pubSubItemsStorageKey(host, name), txn)
		if err != nil {
			return err
		}
		return serializer.DeserializeSlice(val, &items)
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
