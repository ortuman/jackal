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

func (*disabledStorage) UpsertRosterItem(_ *rostermodel.Item) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (*disabledStorage) DeleteRosterItem(_, _ string) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItems(_ string) ([]rostermodel.Item, rostermodel.Version, error) {
	return nil, rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItemsInGroups(_ string, _ []string) ([]rostermodel.Item, rostermodel.Version, error) {
	return nil, rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItem(_, _ string) (*rostermodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) UpsertRosterNotification(_ *rostermodel.Notification) error {
	return nil
}

func (*disabledStorage) DeleteRosterNotification(_, _ string) error {
	return nil
}

func (*disabledStorage) FetchRosterNotification(_ string, _ string) (*rostermodel.Notification, error) {
	return nil, nil
}

func (*disabledStorage) FetchRosterNotifications(_ string) ([]rostermodel.Notification, error) {
	return nil, nil
}

func (*disabledStorage) FetchRosterGroups(_ string) ([]string, error) {
	return nil, nil
}

func (*disabledStorage) InsertOfflineMessage(_ *xmpp.Message, _ string) error {
	return nil
}

func (*disabledStorage) CountOfflineMessages(_ string) (int, error) {
	return 0, nil
}

func (*disabledStorage) FetchOfflineMessages(_ string) ([]xmpp.Message, error) {
	return nil, nil
}

func (*disabledStorage) DeleteOfflineMessages(_ string) error {
	return nil
}

func (*disabledStorage) UpsertVCard(_ context.Context, _ xmpp.XElement, _ string) error {
	return nil
}

func (*disabledStorage) FetchVCard(_ context.Context, _ string) (xmpp.XElement, error) {
	return nil, nil
}

func (*disabledStorage) FetchPrivateXML(_ string, _ string) ([]xmpp.XElement, error) {
	return nil, nil
}

func (*disabledStorage) UpsertPrivateXML(_ []xmpp.XElement, _ string, _ string) error {
	return nil
}

func (*disabledStorage) InsertBlockListItem(_ *model.BlockListItem) error {
	return nil
}

func (*disabledStorage) DeleteBlockListItem(_ *model.BlockListItem) error {
	return nil
}

func (*disabledStorage) FetchBlockListItems(_ string) ([]model.BlockListItem, error) {
	return nil, nil
}

func (*disabledStorage) FetchHosts() (hosts []string, err error) {
	return nil, nil
}

func (*disabledStorage) UpsertNode(_ *pubsubmodel.Node) error {
	return nil
}

func (*disabledStorage) FetchNode(_, _ string) (*pubsubmodel.Node, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodes(_ string) ([]pubsubmodel.Node, error) {
	return nil, nil
}

func (*disabledStorage) FetchSubscribedNodes(_ string) ([]pubsubmodel.Node, error) {
	return nil, nil
}

func (*disabledStorage) DeleteNode(_, _ string) error {
	return nil
}

func (*disabledStorage) UpsertNodeItem(_ *pubsubmodel.Item, _, _ string, _ int) error {
	return nil
}

func (*disabledStorage) FetchNodeItems(_, _ string) ([]pubsubmodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodeItemsWithIDs(_, _ string, _ []string) ([]pubsubmodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodeLastItem(_, _ string) (*pubsubmodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) UpsertNodeAffiliation(_ *pubsubmodel.Affiliation, _, _ string) error {
	return nil
}

func (*disabledStorage) DeleteNodeAffiliation(_, _, _ string) error {
	return nil
}

func (*disabledStorage) FetchNodeAffiliation(_, _, _ string) (*pubsubmodel.Affiliation, error) {
	return nil, nil
}

func (*disabledStorage) FetchNodeAffiliations(_, _ string) ([]pubsubmodel.Affiliation, error) {
	return nil, nil
}

func (*disabledStorage) UpsertNodeSubscription(_ *pubsubmodel.Subscription, _, _ string) error {
	return nil
}

func (*disabledStorage) FetchNodeSubscriptions(_, _ string) ([]pubsubmodel.Subscription, error) {
	return nil, nil
}

func (*disabledStorage) DeleteNodeSubscription(_, _, _ string) error {
	return nil
}

func (*disabledStorage) IsClusterCompatible() bool {
	return false
}

func (*disabledStorage) Close() error {
	return nil
}
