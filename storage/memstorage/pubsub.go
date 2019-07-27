package memstorage

import pubsubmodel "github.com/ortuman/jackal/model/pubsub"

func (m *Storage) UpsertPubSubNode(node *pubsubmodel.Node) error {
	return nil
}

func (m *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	return nil, nil
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
