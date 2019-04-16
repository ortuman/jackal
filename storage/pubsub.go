package storage

import pubsubmodel "github.com/ortuman/jackal/model/pubsub"

type pubSubStorage interface {
	InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error
}
