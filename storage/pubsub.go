package storage

import pubsubmodel "github.com/ortuman/jackal/model/pubsub"

type pubSubStorage interface {
	InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error
}

func InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error {
	return inst.InsertOrUpdatePubSubNode(node)
}
