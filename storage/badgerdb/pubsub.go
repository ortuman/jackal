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
		return b.upsert(node, b.pubSubStorageKey(node.Host, node.Name), tx)
	})
}

func (b *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	var node pubsubmodel.Node
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&node, b.pubSubStorageKey(host, name), txn)
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
	return nil
}

func (b *Storage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	return nil, nil
}

func (b *Storage) UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return nil
}

func (b *Storage) FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	return nil, nil
}

func (b *Storage) pubSubStorageKey(host, name string) []byte {
	return []byte("pubSubNodes:" + host + ":" + name)
}
