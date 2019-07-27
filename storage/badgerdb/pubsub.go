package badgerdb

import (
	"github.com/dgraph-io/badger"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

func (b *Storage) UpsertPubSubNode(node *pubsubmodel.Node) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(node, b.pubSubStorageKey(node.Host, node.Name), tx)
	})
}

func (b *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
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

func (m *Storage) UpsertPubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return nil
}
func (m *Storage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	return nil, nil
}

func (m *Storage) UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return nil
}

func (m *Storage) FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	return nil, nil
}

func (b *Storage) pubSubStorageKey(host, name string) []byte {
	return []byte("pubsub:" + host + ":" + name)
}
