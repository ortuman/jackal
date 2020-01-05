/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
)

type disabledStorage struct{}

func (*disabledStorage) FetchHosts(_ context.Context) (hosts []string, err error) {
	return nil, nil
}

func (*disabledStorage) UpsertNode(_ context.Context, _ *pubsubmodel.Node) error {
	return nil
}

func (*disabledStorage) FetchNode(_ context.Context, _, _ string) (*pubsubmodel.Node, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodes(_ context.Context, _ string) ([]pubsubmodel.Node, error) {
	return nil, nil
}

func (*disabledStorage) FetchSubscribedNodes(_ context.Context, _ string) ([]pubsubmodel.Node, error) {
	return nil, nil
}

func (*disabledStorage) DeleteNode(_ context.Context, _, _ string) error {
	return nil
}

func (*disabledStorage) UpsertNodeItem(_ context.Context, _ *pubsubmodel.Item, _, _ string, _ int) error {
	return nil
}

func (*disabledStorage) FetchNodeItems(_ context.Context, _, _ string) ([]pubsubmodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodeItemsWithIDs(_ context.Context, _, _ string, _ []string) ([]pubsubmodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodeLastItem(_ context.Context, _, _ string) (*pubsubmodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) UpsertNodeAffiliation(_ context.Context, _ *pubsubmodel.Affiliation, _, _ string) error {
	return nil
}

func (*disabledStorage) DeleteNodeAffiliation(_ context.Context, _, _, _ string) error {
	return nil
}

func (*disabledStorage) FetchNodeAffiliation(_ context.Context, _, _, _ string) (*pubsubmodel.Affiliation, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodeAffiliations(_ context.Context, _, _ string) ([]pubsubmodel.Affiliation, error) {
	return nil, nil
}

func (*disabledStorage) UpsertNodeSubscription(_ context.Context, _ *pubsubmodel.Subscription, _, _ string) error {
	return nil
}

func (*disabledStorage) FetchNodeSubscriptions(_ context.Context, _, _ string) ([]pubsubmodel.Subscription, error) {
	return nil, nil
}

func (*disabledStorage) DeleteNodeSubscription(_ context.Context, _, _, _ string) error {
	return nil
}

func (*disabledStorage) IsClusterCompatible() bool {
	return false
}

func (*disabledStorage) Close() error {
	return nil
}
