package storage

import pubsubmodel "github.com/ortuman/jackal/model/pubsub"

type pubSubStorage interface {
	InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error

	GetPubSubNode(host, name string) (*pubsubmodel.Node, error)
}

func InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error {
	return inst.InsertOrUpdatePubSubNode(node)
}

func GetPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	return inst.GetPubSubNode(host, name)
}
