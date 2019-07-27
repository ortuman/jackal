package memstorage

import (
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/model/serializer"
)

func (m *Storage) UpsertPubSubNode(node *pubsubmodel.Node) error {
	b, err := serializer.Serialize(node)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.bytes[pubSubNodeKey(node.Host, node.Name)] = b
		return nil
	})
}

func (m *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[pubSubNodeKey(host, name)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var node pubsubmodel.Node
	if err := serializer.Deserialize(b, &node); err != nil {
		return nil, err
	}
	return &node, nil
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

func pubSubNodeKey(host, name string) string {
	return "pubSubNodes:" + host + ":" + name
}
