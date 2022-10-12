// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"context"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
)

// PubSub defines a repository interface used to store pubsub related data.
type PubSub interface {
	// UpsertNode inserts a node entity into storage, or updates it if was previously inserted.
	UpsertNode(ctx context.Context, node *pubsubmodel.Node) error

	// FetchNode retrieves from storage a node entity associated to a host.
	FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error)

	// FetchNodes retrieves from storage all node entities associated to a host.
	FetchNodes(ctx context.Context, host string) ([]*pubsubmodel.Node, error)

	// NodeExists tells whether a node for a given host exists.
	NodeExists(ctx context.Context, host, name string) (bool, error)

	// DeleteNode deletes a pubsub node from storage.
	DeleteNode(ctx context.Context, host, name string) error

	// DeleteNodes deletes all nodes associated to a host from storage.
	DeleteNodes(ctx context.Context, host string) error

	// UpsertNodeAffiliation inserts a new pubsub node affiliation into storage, or updates it if was previously inserted.
	UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error

	// FetchNodeAffiliation retrieves from storage a node affiliation entity.
	FetchNodeAffiliation(ctx context.Context, jid, host, name string) (*pubsubmodel.Affiliation, error)

	// FetchNodeAffiliations retrieves all affiliations associated to a node.
	FetchNodeAffiliations(ctx context.Context, host, name string) ([]*pubsubmodel.Affiliation, error)

	// DeleteNodeAffiliation deletes a pubsub node affiliation from storage.
	DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error

	// DeleteNodeAffiliations deletes all affiliations associated to a node.
	DeleteNodeAffiliations(ctx context.Context, host, name string) error

	// UpsertNodeSubscription inserts a new pubsub node subscription into storage, or updates it if was previously inserted.
	UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error

	// FetchNodeSubscription retrieves a subscription associated to a node.
	FetchNodeSubscription(ctx context.Context, jid, host, name string) (*pubsubmodel.Subscription, error)

	// FetchNodeSubscriptions retrieves all subscriptions associated to a node.
	FetchNodeSubscriptions(ctx context.Context, host, name string) ([]*pubsubmodel.Subscription, error)

	// DeleteNodeSubscription deletes a pubsub node subscription from storage.
	DeleteNodeSubscription(ctx context.Context, jid, host, name string) error

	// DeleteNodeSubscriptions deletes all subscriptions associated to a node.
	DeleteNodeSubscriptions(ctx context.Context, host, name string) error

	// InsertNodeItem inserts a new pubsub node item into storage.
	InsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string) error

	// FetchNodeItems retrieves all items associated to a node.
	FetchNodeItems(ctx context.Context, host, name string) ([]*pubsubmodel.Item, error)

	// DeleteOldestNodeItems deletes the oldest items associated to a node.
	DeleteOldestNodeItems(ctx context.Context, host, name string, maxItems int) error

	// DeleteNodeItems deletes all items associated to a node.
	DeleteNodeItems(ctx context.Context, host, name string) error
}
