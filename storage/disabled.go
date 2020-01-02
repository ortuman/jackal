/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	"github.com/ortuman/jackal/model"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/xmpp"
)

type disabledStorage struct{}

func (*disabledStorage) UpsertUser(_ context.Context, _ *model.User) error { return nil }
func (*disabledStorage) DeleteUser(_ context.Context, _ string) error      { return nil }
func (*disabledStorage) FetchUser(_ context.Context, _ string) (*model.User, error) {
	return nil, nil
}
func (*disabledStorage) UserExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (*disabledStorage) InsertCapabilities(_ context.Context, _ *model.Capabilities) error {
	return nil
}
func (*disabledStorage) FetchCapabilities(_ context.Context, _, _ string) (*model.Capabilities, error) {
	return nil, nil
}

func (*disabledStorage) UpsertRosterItem(_ context.Context, _ *rostermodel.Item) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (*disabledStorage) DeleteRosterItem(_ context.Context, _, _ string) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItems(_ context.Context, _ string) ([]rostermodel.Item, rostermodel.Version, error) {
	return nil, rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItemsInGroups(_ context.Context, _ string, _ []string) ([]rostermodel.Item, rostermodel.Version, error) {
	return nil, rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItem(_ context.Context, _, _ string) (*rostermodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) UpsertRosterNotification(_ context.Context, _ *rostermodel.Notification) error {
	return nil
}

func (*disabledStorage) DeleteRosterNotification(_ context.Context, _, _ string) error {
	return nil
}

func (*disabledStorage) FetchRosterNotification(_ context.Context, _ string, _ string) (*rostermodel.Notification, error) {
	return nil, nil
}

func (*disabledStorage) FetchRosterNotifications(_ context.Context, _ string) ([]rostermodel.Notification, error) {
	return nil, nil
}

func (*disabledStorage) FetchRosterGroups(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (*disabledStorage) InsertOfflineMessage(_ context.Context, _ *xmpp.Message, _ string) error {
	return nil
}

func (*disabledStorage) CountOfflineMessages(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (*disabledStorage) FetchOfflineMessages(_ context.Context, _ string) ([]xmpp.Message, error) {
	return nil, nil
}

func (*disabledStorage) DeleteOfflineMessages(_ context.Context, _ string) error {
	return nil
}

func (*disabledStorage) FetchPrivateXML(_ context.Context, _ string, _ string) ([]xmpp.XElement, error) {
	return nil, nil
}

func (*disabledStorage) UpsertPrivateXML(_ context.Context, _ []xmpp.XElement, _ string, _ string) error {
	return nil
}

func (*disabledStorage) InsertBlockListItem(_ context.Context, _ *model.BlockListItem) error {
	return nil
}

func (*disabledStorage) DeleteBlockListItem(_ context.Context, _ *model.BlockListItem) error {
	return nil
}

func (*disabledStorage) FetchBlockListItems(_ context.Context, _ string) ([]model.BlockListItem, error) {
	return nil, nil
}

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
