/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

func (b *Storage) UpsertPubSubNode(node *pubsubmodel.Node) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(node, b.pubSubNodesKey(node.Host, node.Name), tx)
	})
}

func (b *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	var node pubsubmodel.Node
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&node, b.pubSubNodesKey(host, name), txn)
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

func (b *Storage) DeletePubSubNode(host, name string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		if err := b.delete(b.pubSubNodesKey(host, name), tx); err != nil {
			return err
		}
		if err := b.delete(b.pubSubItemsKey(host, name), tx); err != nil {
			return err
		}
		return b.delete(b.pubSubAffiliationsKey(host, name), tx)
	})
}

func (b *Storage) UpsertPubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var items []pubsubmodel.Item
		if err := b.fetchSlice(&items, b.pubSubItemsKey(host, name), tx); err != nil {
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
			items = items[len(items)-maxNodeItems:] // remove oldest elements
		}
		return b.upsertSlice(&items, b.pubSubItemsKey(host, name), tx)
	})
}

func (b *Storage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	var items []pubsubmodel.Item
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&items, b.pubSubItemsKey(host, name), txn)
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (b *Storage) UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		var affiliations []pubsubmodel.Affiliation
		if err := b.fetchSlice(&affiliations, b.pubSubAffiliationsKey(host, name), txn); err != nil {
			return err
		}
		affiliations = append(affiliations, *affiliation)
		return b.upsertSlice(&affiliations, b.pubSubAffiliationsKey(host, name), txn)
	})
}

func (b *Storage) FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	var affiliations []pubsubmodel.Affiliation
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&affiliations, b.pubSubAffiliationsKey(host, name), txn)
	})
	if err != nil {
		return nil, err
	}
	return affiliations, nil
}

func (b *Storage) pubSubNodesKey(host, name string) []byte {
	return []byte("pubSubNodes:" + host + ":" + name)
}

func (b *Storage) pubSubItemsKey(host, name string) []byte {
	return []byte("pubSubItems:" + host + ":" + name)
}

func (b *Storage) pubSubAffiliationsKey(host, name string) []byte {
	return []byte("pubSubAffiliations:" + host + ":" + name)
}
