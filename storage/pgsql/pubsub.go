package pgsql

import (
	"errors"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

func (m *Storage) InsertOrUpdatePubSubNode(node *pubsubmodel.Node) error {
	return errors.New("unimplemented method")
}
