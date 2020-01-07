/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

// PubSub defines storage operations for pubsub management.
type PubSub interface {
	// FetchHosts returns all host identifiers.
	FetchHosts(ctx context.Context) (hosts []string, err error)

	// UpsertNode inserts a new pubsub node entity into storage, or updates it if previously inserted.
	UpsertNode(ctx context.Context, node *pubsubmodel.Node) error

	// FetchNode retrieves from storage a pubsub node entity.
	FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error)

	// FetchNodes retrieves from storage all node entities associated with a host.
	FetchNodes(ctx context.Context, host string) ([]pubsubmodel.Node, error)

	// FetchSubscribedNodes retrieves from storage all nodes to which a given jid is subscribed.
	FetchSubscribedNodes(ctx context.Context, jid string) ([]pubsubmodel.Node, error)

	// DeleteNode deletes a pubsub node from storage.
	DeleteNode(ctx context.Context, host, name string) error

	// UpsertNodeItem inserts a new pubsub node item entity into storage, or updates it if previously inserted.
	UpsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string, maxNodeItems int) error

	// FetchNodeItems retrieves all items associated to a node.
	FetchNodeItems(ctx context.Context, host, name string) ([]pubsubmodel.Item, error)

	// FetchNodeItemsWithIDs retrieves all items matching any of the passed identifiers.
	FetchNodeItemsWithIDs(ctx context.Context, host, name string, identifiers []string) ([]pubsubmodel.Item, error)

	// FetchNodeLastItem retrieves last published node item.
	FetchNodeLastItem(ctx context.Context, host, name string) (*pubsubmodel.Item, error)

	// UpsertNodeAffiliation inserts a new pubsub node affiliation into storage, or updates it if previously inserted.
	UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error

	// FetchNodeAffiliation retrieves a concrete node affiliation from storage.
	FetchNodeAffiliation(ctx context.Context, host, name, jid string) (*pubsubmodel.Affiliation, error)

	// FetchNodeAffiliations retrieves all affiliations associated to a node.
	FetchNodeAffiliations(ctx context.Context, host, name string) ([]pubsubmodel.Affiliation, error)

	// DeleteNodeAffiliation deletes a pubsub node affiliation from storage.
	DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error

	// UpsertNodeSubscription inserts a new pubsub node subscription into storage, or updates it if previously inserted.
	UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error

	// FetchNodeSubscriptions retrieves all subscriptions associated to a node.
	FetchNodeSubscriptions(ctx context.Context, host, name string) ([]pubsubmodel.Subscription, error)

	// DeleteNodeSubscription deletes a pubsub node subscription from storage.
	DeleteNodeSubscription(ctx context.Context, jid, host, name string) error
}
