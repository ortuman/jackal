/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"github.com/ortuman/jackal/model"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/xmpp"
)

type disabledStorage struct{}

func (*disabledStorage) UpsertUser(user *model.User) error              { return nil }
func (*disabledStorage) DeleteUser(username string) error               { return nil }
func (*disabledStorage) FetchUser(username string) (*model.User, error) { return nil, nil }
func (*disabledStorage) UserExists(username string) (bool, error)       { return false, nil }

func (*disabledStorage) UpsertRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (*disabledStorage) DeleteRosterItem(username, jid string) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error) {
	return nil, rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItemsInGroups(username string, groups []string) ([]rostermodel.Item, rostermodel.Version, error) {
	return nil, rostermodel.Version{}, nil
}

func (*disabledStorage) FetchRosterItem(username, jid string) (*rostermodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) UpsertRosterNotification(rn *rostermodel.Notification) error {
	return nil
}

func (*disabledStorage) DeleteRosterNotification(contact, jid string) error {
	return nil
}

func (*disabledStorage) FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error) {
	return nil, nil
}

func (*disabledStorage) FetchRosterNotifications(contact string) ([]rostermodel.Notification, error) {
	return nil, nil
}

func (*disabledStorage) InsertOfflineMessage(message *xmpp.Message, username string) error {
	return nil
}

func (*disabledStorage) CountOfflineMessages(username string) (int, error) {
	return 0, nil
}

func (*disabledStorage) FetchOfflineMessages(username string) ([]xmpp.Message, error) {
	return nil, nil
}

func (*disabledStorage) DeleteOfflineMessages(username string) error {
	return nil
}

func (*disabledStorage) UpsertVCard(vCard xmpp.XElement, username string) error {
	return nil
}

func (*disabledStorage) FetchVCard(username string) (xmpp.XElement, error) {
	return nil, nil
}

func (*disabledStorage) FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	return nil, nil
}

func (*disabledStorage) UpsertPrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	return nil
}

func (*disabledStorage) InsertBlockListItem(item *model.BlockListItem) error {
	return nil
}

func (*disabledStorage) DeleteBlockListItem(item *model.BlockListItem) error {
	return nil
}

func (*disabledStorage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	return nil, nil
}

func (*disabledStorage) UpsertPubSubNode(node *pubsubmodel.Node) error {
	return nil
}

func (*disabledStorage) FetchPubSubNode(host, name string) (*pubsubmodel.Node, error) {
	return nil, nil
}

func (*disabledStorage) DeletePubSubNode(host, name string) error {
	return nil
}

func (*disabledStorage) UpsertPubSubNodeItem(item *pubsubmodel.Item, host, name string, maxNodeItems int) error {
	return nil
}

func (*disabledStorage) FetchPubSubNodeItems(host, name string) ([]pubsubmodel.Item, error) {
	return nil, nil
}

func (*disabledStorage) UpsertPubSubNodeAffiliation(affiliation *pubsubmodel.Affiliation, host, name string) error {
	return nil
}

func (*disabledStorage) FetchPubSubNodeAffiliations(host, name string) ([]pubsubmodel.Affiliation, error) {
	return nil, nil
}

func (*disabledStorage) IsClusterCompatible() bool {
	return false
}

func (*disabledStorage) Close() error {
	return nil
}
