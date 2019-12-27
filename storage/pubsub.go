/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

type pubSubStorage interface {
	FetchHosts(ctx context.Context) (hosts []string, err error)

	UpsertNode(ctx context.Context, node *pubsubmodel.Node) error
	FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error)
	FetchNodes(ctx context.Context, host string) ([]pubsubmodel.Node, error)
	FetchSubscribedNodes(ctx context.Context, jid string) ([]pubsubmodel.Node, error)
	DeleteNode(ctx context.Context, host, name string) error

	UpsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string, maxNodeItems int) error
	FetchNodeItems(ctx context.Context, host, name string) ([]pubsubmodel.Item, error)
	FetchNodeItemsWithIDs(ctx context.Context, host, name string, identifiers []string) ([]pubsubmodel.Item, error)
	FetchNodeLastItem(ctx context.Context, host, name string) (*pubsubmodel.Item, error)

	UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error
	FetchNodeAffiliation(ctx context.Context, host, name, jid string) (*pubsubmodel.Affiliation, error)
	FetchNodeAffiliations(ctx context.Context, host, name string) ([]pubsubmodel.Affiliation, error)
	DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error

	UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error
	FetchNodeSubscriptions(ctx context.Context, host, name string) ([]pubsubmodel.Subscription, error)
	DeleteNodeSubscription(ctx context.Context, jid, host, name string) error
}

func FetchHosts(ctx context.Context) (hosts []string, err error) {
	return inst.FetchHosts(ctx)
}

func UpsertNode(ctx context.Context, node *pubsubmodel.Node) error {
	return inst.UpsertNode(ctx, node)
}

func FetchNode(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
	return inst.FetchNode(ctx, host, name)
}

func FetchNodes(ctx context.Context, host string) ([]pubsubmodel.Node, error) {
	return inst.FetchNodes(ctx, host)
}

func FetchSubscribedNodes(ctx context.Context, jid string) ([]pubsubmodel.Node, error) {
	return inst.FetchSubscribedNodes(ctx, jid)
}

func DeleteNode(ctx context.Context, host, name string) error {
	return inst.DeleteNode(ctx, host, name)
}

func UpsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return inst.UpsertNodeItem(ctx, item, host, name, maxNodeItems)
}

func FetchNodeItems(ctx context.Context, host, name string) ([]pubsubmodel.Item, error) {
	return inst.FetchNodeItems(ctx, host, name)
}

func FetchNodeItemsWithIDs(ctx context.Context, host, name string, identifiers []string) ([]pubsubmodel.Item, error) {
	return inst.FetchNodeItemsWithIDs(ctx, host, name, identifiers)
}

func FetchNodeLastItem(ctx context.Context, host, name string) (*pubsubmodel.Item, error) {
	return inst.FetchNodeLastItem(ctx, host, name)
}

func UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
	return inst.UpsertNodeAffiliation(ctx, affiliation, host, name)
}

func DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error {
	return inst.DeleteNodeAffiliation(ctx, jid, host, name)
}

func FetchNodeAffiliation(ctx context.Context, host, name, jid string) (*pubsubmodel.Affiliation, error) {
	return inst.FetchNodeAffiliation(ctx, host, name, jid)
}

func FetchNodeAffiliations(ctx context.Context, host, name string) ([]pubsubmodel.Affiliation, error) {
	return inst.FetchNodeAffiliations(ctx, host, name)
}

func UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
	return inst.UpsertNodeSubscription(ctx, subscription, host, name)
}

func FetchNodeSubscriptions(ctx context.Context, host, name string) ([]pubsubmodel.Subscription, error) {
	return inst.FetchNodeSubscriptions(ctx, host, name)
}

func DeleteNodeSubscription(ctx context.Context, jid, host, name string) error {
	return inst.DeleteNodeSubscription(ctx, jid, host, name)
}
