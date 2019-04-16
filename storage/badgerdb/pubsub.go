package badgerdb

import (
	"github.com/dgraph-io/badger"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

func (b *Storage) InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(node, b.userKey(node.Host+"-"+node.Name), tx)
	})
}
