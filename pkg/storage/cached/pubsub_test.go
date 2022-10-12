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
	"testing"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"

	"github.com/stretchr/testify/require"
)

func TestCachedPubSubRep_UpsertNode(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertNodeFunc = func(ctx context.Context, node *pubsubmodel.Node) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertNode(context.Background(), &pubsubmodel.Node{Host: "h0", Name: "n0"})

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubNodeNS("h0"), cacheNS)
	require.Len(t, repMock.UpsertNodeCalls(), 1)
}

func TestCachedPubSubRep_DeleteNode(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteNodeFunc = func(ctx context.Context, host, node string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteNode(context.Background(), "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubNodeNS("h0"), cacheNS)
	require.Len(t, repMock.DeleteNodeCalls(), 1)
}

func TestCachedPubSubRep_FetchNode(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchNodeFunc = func(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
		return &pubsubmodel.Node{Host: "h0", Name: "n0"}, nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	node, err := rep.FetchNode(context.Background(), "h0", "n0")

	// then
	require.NotNil(t, node)
	require.NoError(t, err)

	require.Equal(t, "h0", node.Host)
	require.Equal(t, "n0", node.Name)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchNodeCalls(), 1)
}

func TestCachedPubSubRep_FetchNodes(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchNodesFunc = func(ctx context.Context, host string) ([]*pubsubmodel.Node, error) {
		return []*pubsubmodel.Node{{Host: "h0", Name: "n0"}}, nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	nodes, err := rep.FetchNodes(context.Background(), "u1")

	// then
	require.NotNil(t, nodes)
	require.NoError(t, err)

	require.Len(t, nodes, 1)
	require.Equal(t, "h0", nodes[0].Host)
	require.Equal(t, "n0", nodes[0].Name)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchNodesCalls(), 1)
}

func TestCachedPubSubRep_DeleteNodes(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteNodesFunc = func(ctx context.Context, host string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteNodes(context.Background(), "h0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubNodeNS("h0"), cacheNS)
	require.Len(t, repMock.DeleteNodesCalls(), 1)
}

func TestCachedPubSubRep_UpsertNodeAffiliation(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertNodeAffiliationFunc = func(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{}, "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubAffiliationNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.UpsertNodeAffiliationCalls(), 1)
}

func TestCachedPubSubRep_FetchNodeAffiliation(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchNodeAffiliationFunc = func(ctx context.Context, jid, host, name string) (*pubsubmodel.Affiliation, error) {
		return &pubsubmodel.Affiliation{Jid: "ortuman@jackal.im"}, nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	affiliation, err := rep.FetchNodeAffiliation(context.Background(), "ortuman@jackal.im", "h0", "n0")

	// then
	require.NotNil(t, affiliation)
	require.NoError(t, err)

	require.Equal(t, "ortuman@jackal.im", affiliation.Jid)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchNodeAffiliationCalls(), 1)
}

func TestCachedPubSubRep_FetchNodeAffiliations(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchNodeAffiliationsFunc = func(ctx context.Context, host, node string) ([]*pubsubmodel.Affiliation, error) {
		return []*pubsubmodel.Affiliation{{Jid: "ortuman@jackal.im"}}, nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	affiliations, err := rep.FetchNodeAffiliations(context.Background(), "h0", "n0")

	// then
	require.NotNil(t, affiliations)
	require.NoError(t, err)

	require.Len(t, affiliations, 1)
	require.Equal(t, "ortuman@jackal.im", affiliations[0].Jid)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchNodeAffiliationsCalls(), 1)
}

func TestCachedPubSubRep_DeleteNodeAffiliation(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteNodeAffiliationFunc = func(ctx context.Context, host, node, jid string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteNodeAffiliation(context.Background(), "ortuman@jackal.im", "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubAffiliationNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.DeleteNodeAffiliationCalls(), 1)
}

func TestCachedPubSubRep_DeleteNodeAffiliations(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteNodeAffiliationsFunc = func(ctx context.Context, host, node string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteNodeAffiliations(context.Background(), "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubAffiliationNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.DeleteNodeAffiliationsCalls(), 1)
}

func TestCachedPubSubRep_UpsertNodeSubscription(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertNodeSubscriptionFunc = func(ctx context.Context, subscription *pubsubmodel.Subscription, host, node string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{}, "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubSubscriptionNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.UpsertNodeSubscriptionCalls(), 1)
}

func TestCachedPubSubRep_FetchNodeSubscription(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchNodeSubscriptionFunc = func(ctx context.Context, jid, host, node string) (*pubsubmodel.Subscription, error) {
		return &pubsubmodel.Subscription{Jid: "ortuman@jackal.im"}, nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	subscription, err := rep.FetchNodeSubscription(context.Background(), "ortuman@jackal.im", "h0", "n0")

	// then
	require.NotNil(t, subscription)
	require.NoError(t, err)

	require.Equal(t, "ortuman@jackal.im", subscription.Jid)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchNodeSubscriptionCalls(), 1)
}

func TestCachedPubSubRep_FetchNodeSubscriptions(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchNodeSubscriptionsFunc = func(ctx context.Context, host, node string) ([]*pubsubmodel.Subscription, error) {
		return []*pubsubmodel.Subscription{{Jid: "ortuman@jackal.im"}}, nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	subscriptions, err := rep.FetchNodeSubscriptions(context.Background(), "h0", "n0")

	// then
	require.NotNil(t, subscriptions)
	require.NoError(t, err)

	require.Len(t, subscriptions, 1)
	require.Equal(t, "ortuman@jackal.im", subscriptions[0].Jid)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchNodeSubscriptionsCalls(), 1)
}

func TestCachedPubSubRep_DeleteNodeSubscription(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteNodeSubscriptionFunc = func(ctx context.Context, host, node, jid string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteNodeSubscription(context.Background(), "ortuman@jackal.im", "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubSubscriptionNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.DeleteNodeSubscriptionCalls(), 1)
}

func TestCachedPubSubRep_DeleteNodeSubscriptions(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteNodeSubscriptionsFunc = func(ctx context.Context, host, node string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteNodeSubscriptions(context.Background(), "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubSubscriptionNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.DeleteNodeSubscriptionsCalls(), 1)
}

func TestCachedPubSubRep_InsertNodeItem(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.InsertNodeItemFunc = func(ctx context.Context, item *pubsubmodel.Item, host, node string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.InsertNodeItem(context.Background(), &pubsubmodel.Item{}, "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubItemNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.InsertNodeItemCalls(), 1)
}

func TestCachedPubSubRep_FetchNodeItems(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchNodeItemsFunc = func(ctx context.Context, host, node string) ([]*pubsubmodel.Item, error) {
		return []*pubsubmodel.Item{{Publisher: "ortuman@jackal.im"}}, nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	items, err := rep.FetchNodeItems(context.Background(), "h0", "n0")

	// then
	require.NotNil(t, items)
	require.NoError(t, err)

	require.Len(t, items, 1)
	require.Equal(t, "ortuman@jackal.im", items[0].Publisher)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchNodeItemsCalls(), 1)
}

func TestCachedPubSubRep_DeleteOldestNodeItems(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteOldestNodeItemsFunc = func(ctx context.Context, host, node string, maxItems int) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteOldestNodeItems(context.Background(), "h0", "n0", 10)

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubItemNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.DeleteOldestNodeItemsCalls(), 1)
}

func TestCachedPubSubRep_DeleteNodeItems(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteNodeItemsFunc = func(ctx context.Context, host, node string) error {
		return nil
	}

	// when
	rep := cachedPubSubRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteNodeItems(context.Background(), "h0", "n0")

	// then
	require.NoError(t, err)
	require.Equal(t, pubSubItemNS("h0", "n0"), cacheNS)
	require.Len(t, repMock.DeleteNodeItemsCalls(), 1)
}
