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

package measuredrepository

import (
	"context"
	"testing"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"

	"github.com/stretchr/testify/require"
)

func TestMeasuredPubSubRep_UpsertNode(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertNodeFunc = func(ctx context.Context, node *pubsubmodel.Node) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.UpsertNode(context.Background(), &pubsubmodel.Node{})

	// then
	require.Nil(t, err)

	require.Len(t, repMock.UpsertNodeCalls(), 1)
}

func TestMeasuredPubSubRep_FetchNode(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchNodeFunc = func(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, err := m.FetchNode(context.Background(), "h0", "n0")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.FetchNodeCalls(), 1)
}

func TestMeasuredPubSubRep_FetchNodes(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchNodesFunc = func(ctx context.Context, host string) ([]*pubsubmodel.Node, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, err := m.FetchNodes(context.Background(), "n0")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.FetchNodesCalls(), 1)
}

func TestMeasuredPubSubRep_NodeExists(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
		return true, nil
	}
	m := New(repMock)

	// when
	_, _ = m.NodeExists(context.Background(), "h0", "n0")

	// then
	require.Len(t, repMock.NodeExistsCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteNode(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteNodeFunc = func(ctx context.Context, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteNode(context.Background(), "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteNodeCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteNodes(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteNodesFunc = func(ctx context.Context, host string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteNodes(context.Background(), "n0")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteNodesCalls(), 1)
}

func TestMeasuredPubSubRep_UpsertNodeAffiliation(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertNodeAffiliationFunc = func(ctx context.Context, affiliation *pubsubmodel.Affiliation, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{}, "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.UpsertNodeAffiliationCalls(), 1)
}

func TestMeasuredPubSubRep_FetchNodeAffiliation(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchNodeAffiliationFunc = func(ctx context.Context, jid string, host string, name string) (*pubsubmodel.Affiliation, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, err := m.FetchNodeAffiliation(context.Background(), "ortuman@jackal.im", "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.FetchNodeAffiliationCalls(), 1)
}

func TestMeasuredPubSubRep_FetchNodeAffiliations(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchNodeAffiliationsFunc = func(ctx context.Context, host string, name string) ([]*pubsubmodel.Affiliation, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, err := m.FetchNodeAffiliations(context.Background(), "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.FetchNodeAffiliationsCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteNodeAffiliation(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteNodeAffiliationFunc = func(ctx context.Context, jid string, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteNodeAffiliation(context.Background(), "ortuman@jackal.im", "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteNodeAffiliationCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteNodeAffiliations(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteNodeAffiliationsFunc = func(ctx context.Context, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteNodeAffiliations(context.Background(), "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteNodeAffiliationsCalls(), 1)
}

func TestMeasuredPubSubRep_UpsertNodeSubscription(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertNodeSubscriptionFunc = func(ctx context.Context, subscription *pubsubmodel.Subscription, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{}, "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.UpsertNodeSubscriptionCalls(), 1)
}

func TestMeasuredPubSubRep_FetchNodeSubscription(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchNodeSubscriptionFunc = func(ctx context.Context, jid string, host string, name string) (*pubsubmodel.Subscription, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, err := m.FetchNodeSubscription(context.Background(), "ortuman@jackal.im", "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.FetchNodeSubscriptionCalls(), 1)
}

func TestMeasuredPubSubRep_FetchNodeSubscriptions(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchNodeSubscriptionsFunc = func(ctx context.Context, host string, name string) ([]*pubsubmodel.Subscription, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, err := m.FetchNodeSubscriptions(context.Background(), "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.FetchNodeSubscriptionsCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteNodeSubscription(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteNodeSubscriptionFunc = func(ctx context.Context, jid string, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteNodeSubscription(context.Background(), "ortuman@jackal.im", "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteNodeSubscriptionCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteNodeSubscriptions(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteNodeSubscriptionsFunc = func(ctx context.Context, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteNodeSubscriptions(context.Background(), "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteNodeSubscriptionsCalls(), 1)
}

func TestMeasuredPubSubRep_InsertNodeItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.InsertNodeItemFunc = func(ctx context.Context, item *pubsubmodel.Item, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.InsertNodeItem(context.Background(), &pubsubmodel.Item{}, "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.InsertNodeItemCalls(), 1)
}

func TestMeasuredPubSubRep_FetchNodeItems(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchNodeItemsFunc = func(ctx context.Context, host string, name string) ([]*pubsubmodel.Item, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, err := m.FetchNodeItems(context.Background(), "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.FetchNodeItemsCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteOldestNodeItems(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteOldestNodeItemsFunc = func(ctx context.Context, host string, name string, maxItems int) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteOldestNodeItems(context.Background(), "n0", "blogs", 10)

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteOldestNodeItemsCalls(), 1)
}

func TestMeasuredPubSubRep_DeleteNodeItems(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteNodeItemsFunc = func(ctx context.Context, host string, name string) error {
		return nil
	}
	m := New(repMock)

	// when
	err := m.DeleteNodeItems(context.Background(), "n0", "blogs")

	// then
	require.Nil(t, err)

	require.Len(t, repMock.DeleteNodeItemsCalls(), 1)
}
