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
		m.bytes[pubSubNodesKey(node.Host, node.Name)] = b
		return m.upsertHostNode(node)
	})
}

func (m *Storage) FetchPubSubNodes(host string) ([]pubsubmodel.Node, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[pubSubHostNodesKey(host)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var nodes []pubsubmodel.Node

	if err := serializer.DeserializeSlice(b, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (m *Storage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[pubSubNodesKey(host, name)]
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

func (m *Storage) FetchPubSubSubscribedNodes(jid string) ([]pubsubmodel.Node, error) {
	// TODO(ortuman): implement me!
	return nil, nil
}

func (m *Storage) DeletePubSubNode(host, name string) error {
	return m.inWriteLock(func() error {
		delete(m.bytes, pubSubNodesKey(host, name))
		delete(m.bytes, pubSubItemsKey(host, name))
		delete(m.bytes, pubSubAffiliationsKey(host, name))
		return m.deleteHostNode(host, name)
	})
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

func (m *Storage) FetchPubSubNodeItemsWithIDs(host, name string, identifiers []string) ([]pubsubmodel.Item, error) {
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
	identifiersSet := make(map[string]struct{})
	for _, id := range identifiers {
		identifiersSet[id] = struct{}{}
	}
	var filteredItems, items []pubsubmodel.Item
	if err := serializer.DeserializeSlice(b, &items); err != nil {
		return nil, err
	}
	for _, itm := range items {
		if _, ok := identifiersSet[itm.ID]; ok {
			filteredItems = append(filteredItems, itm)
		}
	}
	return filteredItems, nil
}

func (m *Storage) FetchPubSubNodeLastItem(host, name string) (*pubsubmodel.Item, error) {
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
	return &items[len(items)-1], nil
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

func (m *Storage) DeletePubSubNodeAffiliation(jid, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var affiliations []pubsubmodel.Affiliation

		b = m.bytes[pubSubAffiliationsKey(host, name)]
		if b != nil {
			if err := serializer.DeserializeSlice(b, &affiliations); err != nil {
				return err
			}
		}
		var deleted bool
		for i, aff := range affiliations {
			if aff.JID == jid {
				affiliations = append(affiliations[:i], affiliations[i+1:]...)
				deleted = true
				break
			}
		}
		if !deleted {
			return nil
		}
		b, err := serializer.SerializeSlice(&affiliations)
		if err != nil {
			return err
		}
		m.bytes[pubSubAffiliationsKey(host, name)] = b
		return nil
	})
}

func (m *Storage) UpsertPubSubNodeSubscription(subscription *pubsubmodel.Subscription, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var subscriptions []pubsubmodel.Subscription

		b = m.bytes[pubSubSubscriptionsKey(host, name)]
		if b != nil {
			if err := serializer.DeserializeSlice(b, &subscriptions); err != nil {
				return err
			}
		}
		var updated bool
		for i, sub := range subscriptions {
			if sub.JID == subscription.JID {
				subscriptions[i] = *subscription
				updated = true
				break
			}
		}
		if !updated {
			subscriptions = append(subscriptions, *subscription)
		}
		b, err := serializer.SerializeSlice(&subscriptions)
		if err != nil {
			return err
		}
		m.bytes[pubSubSubscriptionsKey(host, name)] = b
		return nil
	})
}

func (m *Storage) FetchPubSubNodeSubscriptions(host, name string) ([]pubsubmodel.Subscription, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[pubSubSubscriptionsKey(host, name)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var subscriptions []pubsubmodel.Subscription
	if err := serializer.DeserializeSlice(b, &subscriptions); err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (m *Storage) DeletePubSubNodeSubscription(jid, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var subscriptions []pubsubmodel.Subscription

		b = m.bytes[pubSubSubscriptionsKey(host, name)]
		if b != nil {
			if err := serializer.DeserializeSlice(b, &subscriptions); err != nil {
				return err
			}
		}
		var deleted bool
		for i, sub := range subscriptions {
			if sub.JID == jid {
				subscriptions = append(subscriptions[:i], subscriptions[i+1:]...)
				deleted = true
				break
			}
		}
		if !deleted {
			return nil
		}
		b, err := serializer.SerializeSlice(&subscriptions)
		if err != nil {
			return err
		}
		m.bytes[pubSubSubscriptionsKey(host, name)] = b
		return nil
	})
}

func (m *Storage) upsertHostNode(node *pubsubmodel.Node) error {
	var nodes []pubsubmodel.Node

	b := m.bytes[pubSubHostNodesKey(node.Host)]
	if b != nil {
		if err := serializer.DeserializeSlice(b, &nodes); err != nil {
			return err
		}
	}
	var updated bool

	for i, n := range nodes {
		if n.Name == node.Name {
			nodes[i] = *node
			updated = true
			break
		}
	}
	if !updated {
		nodes = append(nodes, *node)
	}

	b, err := serializer.SerializeSlice(&nodes)
	if err != nil {
		return err
	}
	m.bytes[pubSubHostNodesKey(node.Host)] = b
	return nil
}

func (m *Storage) deleteHostNode(host, name string) error {
	var nodes []pubsubmodel.Node

	b := m.bytes[pubSubHostNodesKey(host)]
	if b != nil {
		if err := serializer.DeserializeSlice(b, &nodes); err != nil {
			return err
		}
	}
	for i, n := range nodes {
		if n.Name == name {
			nodes = append(nodes[:i], nodes[i+1:]...)
			break
		}
	}

	b, err := serializer.SerializeSlice(&nodes)
	if err != nil {
		return err
	}
	m.bytes[pubSubHostNodesKey(host)] = b
	return nil
}

func pubSubHostNodesKey(host string) string {
	return "pubSubHostNodes:" + host
}

func pubSubNodesKey(host, name string) string {
	return "pubSubNodes:" + host + ":" + name
}

func pubSubAffiliationsKey(host, name string) string {
	return "pubSubAffiliations:" + host + ":" + name
}

func pubSubSubscriptionsKey(host, name string) string {
	return "pubSubSubscriptions:" + host + ":" + name
}

func pubSubItemsKey(host, name string) string {
	return "pubSubItems:" + host + ":" + name
}
