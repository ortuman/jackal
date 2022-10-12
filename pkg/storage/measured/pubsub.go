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
	"time"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type measuredPubSubRep struct {
	rep  repository.PubSub
	inTx bool
}

func (m *measuredPubSubRep) UpsertNode(ctx context.Context, node *pubsubmodel.Node) error {
	t0 := time.Now()
	err := m.rep.UpsertNode(ctx, node)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) FetchNode(ctx context.Context, host, name string) (node *pubsubmodel.Node, err error) {
	t0 := time.Now()
	node, err = m.rep.FetchNode(ctx, host, name)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) FetchNodes(ctx context.Context, host string) (nodes []*pubsubmodel.Node, err error) {
	t0 := time.Now()
	nodes, err = m.rep.FetchNodes(ctx, host)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) NodeExists(ctx context.Context, host, name string) (ok bool, err error) {
	t0 := time.Now()
	ok, err = m.rep.NodeExists(ctx, host, name)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) DeleteNode(ctx context.Context, host, name string) error {
	t0 := time.Now()
	err := m.rep.DeleteNode(ctx, host, name)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) DeleteNodes(ctx context.Context, host string) error {
	t0 := time.Now()
	err := m.rep.DeleteNodes(ctx, host)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
	t0 := time.Now()
	err := m.rep.UpsertNodeAffiliation(ctx, affiliation, host, name)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) FetchNodeAffiliation(ctx context.Context, jid, host, name string) (affiliation *pubsubmodel.Affiliation, err error) {
	t0 := time.Now()
	affiliation, err = m.rep.FetchNodeAffiliation(ctx, jid, host, name)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) FetchNodeAffiliations(ctx context.Context, host, name string) (affiliations []*pubsubmodel.Affiliation, err error) {
	t0 := time.Now()
	affiliations, err = m.rep.FetchNodeAffiliations(ctx, host, name)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error {
	t0 := time.Now()
	err := m.rep.DeleteNodeAffiliation(ctx, jid, host, name)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) DeleteNodeAffiliations(ctx context.Context, host, name string) error {
	t0 := time.Now()
	err := m.rep.DeleteNodeAffiliations(ctx, host, name)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
	t0 := time.Now()
	err := m.rep.UpsertNodeSubscription(ctx, subscription, host, name)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) FetchNodeSubscription(ctx context.Context, jid, host, name string) (subscription *pubsubmodel.Subscription, err error) {
	t0 := time.Now()
	subscription, err = m.rep.FetchNodeSubscription(ctx, jid, host, name)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) FetchNodeSubscriptions(ctx context.Context, host, name string) (subscriptions []*pubsubmodel.Subscription, err error) {
	t0 := time.Now()
	subscriptions, err = m.rep.FetchNodeSubscriptions(ctx, host, name)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) DeleteNodeSubscription(ctx context.Context, jid, host, name string) error {
	t0 := time.Now()
	err := m.rep.DeleteNodeSubscription(ctx, jid, host, name)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) DeleteNodeSubscriptions(ctx context.Context, host, name string) error {
	t0 := time.Now()
	err := m.rep.DeleteNodeSubscriptions(ctx, host, name)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) InsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string) error {
	t0 := time.Now()
	err := m.rep.InsertNodeItem(ctx, item, host, name)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) FetchNodeItems(ctx context.Context, host, name string) (items []*pubsubmodel.Item, err error) {
	t0 := time.Now()
	items, err = m.rep.FetchNodeItems(ctx, host, name)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredPubSubRep) DeleteOldestNodeItems(ctx context.Context, host, name string, maxItems int) error {
	t0 := time.Now()
	err := m.rep.DeleteOldestNodeItems(ctx, host, name, maxItems)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredPubSubRep) DeleteNodeItems(ctx context.Context, host, name string) error {
	t0 := time.Now()
	err := m.rep.DeleteNodeItems(ctx, host, name)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}
