/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

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
	return m.inWriteLock(func() error {
		var b []byte
		var items []pubsubmodel.Item

		b = m.bytes[pubSubItemsKey(host, name)]
		if b != nil {
			if err := serializer.DeserializeSlice(b, &items); err != nil {
				return err
			}
		}
		var updated bool
		for i, itm := range items {
			if itm.ID == item.ID {
				items[i] = *item
				updated = true
				break
			}
		}
		if !updated {
			items = append(items, *item)
		}
		if len(items) > maxNodeItems {
			items = items[len(items)-maxNodeItems:] // remove oldest elements
		}
		b, err := serializer.SerializeSlice(&items)
		if err != nil {
			return err
		}
		m.bytes[pubSubItemsKey(host, name)] = b
		return nil
	})
}

func (m *Storage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[pubSubItemsKey(host, name)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var items []pubsubmodel.Item
	if err := serializer.DeserializeSlice(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (m *Storage) UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var affiliations []pubsubmodel.Affiliation

		b = m.bytes[pubSubAffiliationsKey(host, name)]
		if b != nil {
			if err := serializer.DeserializeSlice(b, &affiliations); err != nil {
				return err
			}
		}
		var updated bool
		for i, aff := range affiliations {
			if aff.JID == affiliation.JID {
				affiliations[i] = *affiliation
				updated = true
				break
			}
		}
		if !updated {
			affiliations = append(affiliations, *affiliation)
		}
		b, err := serializer.SerializeSlice(&affiliations)
		if err != nil {
			return err
		}
		m.bytes[pubSubAffiliationsKey(host, name)] = b
		return nil
	})
}

func (m *Storage) FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[pubSubAffiliationsKey(host, name)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var affiliations []pubsubmodel.Affiliation
	if err := serializer.DeserializeSlice(b, &affiliations); err != nil {
		return nil, err
	}
	return affiliations, nil
}

func pubSubNodeKey(host, name string) string {
	return "pubSubNodes:" + host + ":" + name
}

func pubSubAffiliationsKey(host, name string) string {
	return "pubSubAffiliations:" + host + ":" + name
}

func pubSubItemsKey(host, name string) string {
	return "pubSubItems:" + host + ":" + name
}
