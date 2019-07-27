/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/xmpp"
)

type disabledStorage struct{}

func (*disabledStorage) InsertOrUpdateUser(user *model.User) error      { return nil }
func (*disabledStorage) DeleteUser(username string) error               { return nil }
func (*disabledStorage) FetchUser(username string) (*model.User, error) { return nil, nil }
func (*disabledStorage) UserExists(username string) (bool, error)       { return false, nil }

func (*disabledStorage) InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
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

func (*disabledStorage) InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error {
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

func (*disabledStorage) InsertOrUpdateVCard(vCard xmpp.XElement, username string) error {
	return nil
}

func (*disabledStorage) FetchVCard(username string) (xmpp.XElement, error) {
	return nil, nil
}

func (*disabledStorage) FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	return nil, nil
}

func (*disabledStorage) InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	return nil
}

func (*disabledStorage) InsertBlockListItems(items []model.BlockListItem) error {
	return nil
}

func (*disabledStorage) DeleteBlockListItems(items []model.BlockListItem) error {
	return nil
}

func (*disabledStorage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	return nil, nil
}

func (*disabledStorage) IsClusterCompatible() bool {
	return false
}

func (*disabledStorage) Close() error {
	return nil
}
