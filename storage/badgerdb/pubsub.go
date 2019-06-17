package badgerdb

import (
	"github.com/dgraph-io/badger"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

func (b *Storage) InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(node, b.pubSubStorageKey(node.Host, node.Name), tx)
	})
}

func (b *Storage) GetPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	var node pubsubmodel.Node
	err := b.fetch(&node, b.pubSubStorageKey(host, name))
	switch err {
	case nil:
		return &node, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *Storage) pubSubStorageKey(host, name string) []byte {
	return []byte("pubsub:" + host + ":" + name)
}
