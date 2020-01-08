/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"strings"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/model/serializer"
)

// PubSub represents an in-memory pubsub storage.
type PubSub struct {
	*memoryStorage
}

// NewPubSub returns an instance of PubSub in-memory storage.
func NewPubSub() *PubSub {
	return &PubSub{memoryStorage: newStorage()}
}

// FetchHosts returns all host identifiers.
func (m *PubSub) FetchHosts(_ context.Context) ([]string, error) {
	var hosts []string
	if err := m.inReadLock(func() error {
		for k := range m.b {
			if !strings.HasPrefix(k, "pubSubHostNodes:") {
				continue
			}
			keySplits := strings.Split(k, ":")
			if len(keySplits) != 2 {
				continue
			}
			host := keySplits[1]

			var isPresent bool
			for _, h := range hosts {
				if h == host {
					isPresent = true
					break
				}
			}
			if isPresent {
				continue
			}
			hosts = append(hosts, host)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return hosts, nil
}

// UpsertNode inserts a new pubsub node entity into storage, or updates it if previously inserted.
func (m *PubSub) UpsertNode(_ context.Context, node *pubsubmodel.Node) error {
	b, err := serializer.Serialize(node)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.b[pubSubNodesKey(node.Host, node.Name)] = b
		return m.upsertHostNode(node)
	})
}

// FetchNodes retrieves from storage all node entities associated with a host.
func (m *PubSub) FetchNodes(_ context.Context, host string) ([]pubsubmodel.Node, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[pubSubHostNodesKey(host)]
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

// FetchNode retrieves from storage a pubsub node entity.
func (m *PubSub) FetchNode(_ context.Context, host, name string) (*pubsubmodel.Node, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[pubSubNodesKey(host, name)]
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

// FetchSubscribedNodes retrieves from storage all nodes to which a given jid is subscribed.
func (m *PubSub) FetchSubscribedNodes(_ context.Context, jid string) ([]pubsubmodel.Node, error) {
	var nodes []pubsubmodel.Node
	if err := m.inReadLock(func() error {
		for k, b := range m.b {
			if !strings.HasPrefix(k, "pubSubSubscriptions:") {
				continue
			}
			keySplits := strings.Split(k, ":")
			if len(keySplits) != 3 {
				continue // wrong key format
			}
			host := keySplits[1]
			name := keySplits[2]

			var subs []pubsubmodel.Subscription
			if b != nil {
				if err := serializer.DeserializeSlice(b, &subs); err != nil {
					return err
				}
			}
			for _, sub := range subs {
				if sub.JID != jid || sub.Subscription != pubsubmodel.Subscribed {
					continue
				}
				// fetch pubsub node
				var node pubsubmodel.Node

				b := m.b[pubSubNodesKey(host, name)]
				if b == nil {
					continue
				}
				if err := serializer.Deserialize(b, &node); err != nil {
					return err
				}
				nodes = append(nodes, node)
				break
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return nodes, nil
}

// DeleteNode deletes a pubsub node from storage.
func (m *PubSub) DeleteNode(_ context.Context, host, name string) error {
	return m.inWriteLock(func() error {
		delete(m.b, pubSubNodesKey(host, name))
		delete(m.b, pubSubItemsKey(host, name))
		delete(m.b, pubSubAffiliationsKey(host, name))
		return m.deleteHostNode(host, name)
	})
}

// UpsertNodeItem inserts a new pubsub node item entity into storage, or updates it if previously inserted.
func (m *PubSub) UpsertNodeItem(_ context.Context, item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return m.inWriteLock(func() error {
		var b []byte
		var items []pubsubmodel.Item

		b = m.b[pubSubItemsKey(host, name)]
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
		m.b[pubSubItemsKey(host, name)] = b
		return nil
	})
}

// FetchNodeItems retrieves all items associated to a node.
func (m *PubSub) FetchNodeItems(_ context.Context, host, name string) ([]pubsubmodel.Item, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[pubSubItemsKey(host, name)]
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

// FetchNodeItemsWithIDs retrieves all items matching any of the passed identifiers.
func (m *PubSub) FetchNodeItemsWithIDs(_ context.Context, host, name string, identifiers []string) ([]pubsubmodel.Item, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[pubSubItemsKey(host, name)]
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

// FetchNodeLastItem retrieves last published node item.
func (m *PubSub) FetchNodeLastItem(_ context.Context, host, name string) (*pubsubmodel.Item, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[pubSubItemsKey(host, name)]
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

// UpsertNodeAffiliation inserts a new pubsub node affiliation into storage, or updates it if previously inserted.
func (m *PubSub) UpsertNodeAffiliation(_ context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var affiliations []pubsubmodel.Affiliation

		b = m.b[pubSubAffiliationsKey(host, name)]
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
		m.b[pubSubAffiliationsKey(host, name)] = b
		return nil
	})
}

// FetchNodeAffiliation retrieves a concrete node affiliation from storage.
func (m *PubSub) FetchNodeAffiliation(ctx context.Context, host, name, jid string) (*pubsubmodel.Affiliation, error) {
	affiliations, err := m.FetchNodeAffiliations(ctx, host, name)
	if err != nil {
		return nil, err
	}
	for _, aff := range affiliations {
		if aff.JID == jid {
			return &aff, nil
		}
	}
	return nil, nil
}

// FetchNodeAffiliations retrieves all affiliations associated to a node.
func (m *PubSub) FetchNodeAffiliations(_ context.Context, host, name string) ([]pubsubmodel.Affiliation, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[pubSubAffiliationsKey(host, name)]
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

// DeleteNodeAffiliation deletes a pubsub node affiliation from storage.
func (m *PubSub) DeleteNodeAffiliation(_ context.Context, jid, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var affiliations []pubsubmodel.Affiliation

		b = m.b[pubSubAffiliationsKey(host, name)]
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
		m.b[pubSubAffiliationsKey(host, name)] = b
		return nil
	})
}

// UpsertNodeSubscription inserts a new pubsub node subscription into storage, or updates it if previously inserted.
func (m *PubSub) UpsertNodeSubscription(_ context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var subscriptions []pubsubmodel.Subscription

		b = m.b[pubSubSubscriptionsKey(host, name)]
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
		m.b[pubSubSubscriptionsKey(host, name)] = b
		return nil
	})
}

// FetchNodeSubscriptions retrieves all subscriptions associated to a node.
func (m *PubSub) FetchNodeSubscriptions(_ context.Context, host, name string) ([]pubsubmodel.Subscription, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[pubSubSubscriptionsKey(host, name)]
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

// DeleteNodeSubscription deletes a pubsub node subscription from storage.
func (m *PubSub) DeleteNodeSubscription(_ context.Context, jid, host, name string) error {
	return m.inWriteLock(func() error {
		var b []byte
		var subscriptions []pubsubmodel.Subscription

		b = m.b[pubSubSubscriptionsKey(host, name)]
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
		m.b[pubSubSubscriptionsKey(host, name)] = b
		return nil
	})
}

func (m *PubSub) upsertHostNode(node *pubsubmodel.Node) error {
	var nodes []pubsubmodel.Node

	b := m.b[pubSubHostNodesKey(node.Host)]
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
	m.b[pubSubHostNodesKey(node.Host)] = b
	return nil
}

func (m *PubSub) deleteHostNode(host, name string) error {
	var nodes []pubsubmodel.Node

	b := m.b[pubSubHostNodesKey(host)]
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
	m.b[pubSubHostNodesKey(host)] = b
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
