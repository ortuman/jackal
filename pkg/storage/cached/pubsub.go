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

package cachedrepository

import (
	"context"

	"github.com/go-kit/log"

	"github.com/ortuman/jackal/pkg/model"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const (
	pubSubNodesKey         = "nodes"
	pubSubAffiliationsKey  = "affiliations"
	pubSubSubscriptionsKey = "subscriptions"
	pubSubItemsKey         = "items"
)

type cachedPubSubRep struct {
	c      Cache
	rep    repository.PubSub
	logger log.Logger
}

func (c *cachedPubSubRep) UpsertNode(ctx context.Context, node *pubsubmodel.Node) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubNodeNS(node.Host),
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertNode(ctx, node)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
	op := fetchOp{
		c:         c.c,
		namespace: pubSubNodeNS(host),
		key:       name,
		codec:     &pubsubmodel.Node{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return c.rep.FetchNode(ctx, host, name)
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*pubsubmodel.Node), nil
	}
	return nil, nil
}

func (c *cachedPubSubRep) FetchNodes(ctx context.Context, host string) ([]*pubsubmodel.Node, error) {
	op := fetchOp{
		c:         c.c,
		namespace: pubSubNodeNS(host),
		key:       pubSubNodesKey,
		codec:     &pubsubmodel.Nodes{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			nodes, err := c.rep.FetchNodes(ctx, host)
			if err != nil {
				return nil, err
			}
			return &pubsubmodel.Nodes{Nodes: nodes}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*pubsubmodel.Nodes).Nodes, nil
	}
	return nil, nil
}

func (c *cachedPubSubRep) NodeExists(ctx context.Context, host, name string) (bool, error) {
	op := existsOp{
		c:         c.c,
		namespace: pubSubNodeNS(host),
		key:       name,
		missFn: func(ctx context.Context) (bool, error) {
			return c.rep.NodeExists(ctx, host, name)
		},
		logger: c.logger,
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) DeleteNode(ctx context.Context, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubNodeNS(host),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteNode(ctx, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) DeleteNodes(ctx context.Context, host string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubNodeNS(host),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteNodes(ctx, host)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubAffiliationNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertNodeAffiliation(ctx, affiliation, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) FetchNodeAffiliation(ctx context.Context, jid, host, name string) (*pubsubmodel.Affiliation, error) {
	op := fetchOp{
		c:         c.c,
		namespace: pubSubAffiliationNS(host, name),
		key:       jid,
		codec:     &pubsubmodel.Affiliation{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return c.rep.FetchNodeAffiliation(ctx, jid, host, name)
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*pubsubmodel.Affiliation), nil
	}
	return nil, nil
}

func (c *cachedPubSubRep) FetchNodeAffiliations(ctx context.Context, host, name string) ([]*pubsubmodel.Affiliation, error) {
	op := fetchOp{
		c:         c.c,
		namespace: pubSubAffiliationNS(host, name),
		key:       pubSubAffiliationsKey,
		codec:     &pubsubmodel.Affiliations{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			affiliations, err := c.rep.FetchNodeAffiliations(ctx, host, name)
			if err != nil {
				return nil, err
			}
			return &pubsubmodel.Affiliations{Affiliations: affiliations}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*pubsubmodel.Affiliations).Affiliations, nil
	}
	return nil, nil
}

func (c *cachedPubSubRep) DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubAffiliationNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteNodeAffiliation(ctx, jid, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) DeleteNodeAffiliations(ctx context.Context, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubAffiliationNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteNodeAffiliations(ctx, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubSubscriptionNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertNodeSubscription(ctx, subscription, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) FetchNodeSubscription(ctx context.Context, jid, host, name string) (*pubsubmodel.Subscription, error) {
	op := fetchOp{
		c:         c.c,
		namespace: pubSubSubscriptionNS(host, name),
		key:       jid,
		codec:     &pubsubmodel.Subscription{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return c.rep.FetchNodeSubscription(ctx, jid, host, name)
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*pubsubmodel.Subscription), nil
	}
	return nil, nil
}

func (c *cachedPubSubRep) FetchNodeSubscriptions(ctx context.Context, host, name string) ([]*pubsubmodel.Subscription, error) {
	op := fetchOp{
		c:         c.c,
		namespace: pubSubSubscriptionNS(host, name),
		key:       pubSubSubscriptionsKey,
		codec:     &pubsubmodel.Subscriptions{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			subscriptions, err := c.rep.FetchNodeSubscriptions(ctx, host, name)
			if err != nil {
				return nil, err
			}
			return &pubsubmodel.Subscriptions{Subscriptions: subscriptions}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*pubsubmodel.Subscriptions).Subscriptions, nil
	}
	return nil, nil
}

func (c *cachedPubSubRep) DeleteNodeSubscription(ctx context.Context, jid, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubSubscriptionNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteNodeSubscription(ctx, jid, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) DeleteNodeSubscriptions(ctx context.Context, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubSubscriptionNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteNodeSubscriptions(ctx, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) InsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubItemNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.InsertNodeItem(ctx, item, host, name)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) FetchNodeItems(ctx context.Context, host, name string) ([]*pubsubmodel.Item, error) {
	op := fetchOp{
		c:         c.c,
		namespace: pubSubItemNS(host, name),
		key:       pubSubItemsKey,
		codec:     &pubsubmodel.Items{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			items, err := c.rep.FetchNodeItems(ctx, host, name)
			if err != nil {
				return nil, err
			}
			return &pubsubmodel.Items{Items: items}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*pubsubmodel.Items).Items, nil
	}
	return nil, nil
}

func (c *cachedPubSubRep) DeleteOldestNodeItems(ctx context.Context, host, name string, maxItems int) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubItemNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteOldestNodeItems(ctx, host, name, maxItems)
		},
	}
	return op.do(ctx)
}

func (c *cachedPubSubRep) DeleteNodeItems(ctx context.Context, host, name string) error {
	op := updateOp{
		c:         c.c,
		namespace: pubSubItemNS(host, name),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteNodeItems(ctx, host, name)
		},
	}
	return op.do(ctx)
}

func pubSubNodeNS(host string) string {
	return "pubsub:nodes:" + host
}

func pubSubAffiliationNS(host, name string) string {
	return "pubsub:affiliations:" + host + ":" + name
}

func pubSubSubscriptionNS(host, name string) string {
	return "pubsub:subscriptions:" + host + ":" + name
}

func pubSubItemNS(host, name string) string {
	return "pubsub:items:" + host + ":" + name
}
