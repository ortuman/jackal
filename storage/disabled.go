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

func (_ *disabledStorage) IsClusterCompatible() bool { return false }

func (_ *disabledStorage) InsertOrUpdateUser(user *model.User) error      { return nil }
func (_ *disabledStorage) DeleteUser(username string) error               { return nil }
func (_ *disabledStorage) FetchUser(username string) (*model.User, error) { return nil, nil }
func (_ *disabledStorage) UserExists(username string) (bool, error)       { return false, nil }

func (_ *disabledStorage) InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (_ *disabledStorage) DeleteRosterItem(username, jid string) (rostermodel.Version, error) {
	return rostermodel.Version{}, nil
}

func (_ *disabledStorage) FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error) {
	return nil, rostermodel.Version{}, nil
}

func (_ *disabledStorage) FetchRosterItem(username, jid string) (*rostermodel.Item, error) {
	return nil, nil
}

func (_ *disabledStorage) InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error {
	return nil
}

func (_ *disabledStorage) DeleteRosterNotification(contact, jid string) error {
	return nil
}

func (_ *disabledStorage) FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error) {
	return nil, nil
}

func (_ *disabledStorage) FetchRosterNotifications(contact string) ([]rostermodel.Notification, error) {
	return nil, nil
}

func (_ *disabledStorage) InsertOfflineMessage(message *xmpp.Message, username string) error {
	return nil
}

func (_ *disabledStorage) CountOfflineMessages(username string) (int, error) {
	return 0, nil
}

func (_ *disabledStorage) FetchOfflineMessages(username string) ([]*xmpp.Message, error) {
	return nil, nil
}

func (_ *disabledStorage) DeleteOfflineMessages(username string) error {
	return nil
}

func (_ *disabledStorage) InsertOrUpdateVCard(vCard xmpp.XElement, username string) error {
	return nil
}

func (_ *disabledStorage) FetchVCard(username string) (xmpp.XElement, error) {
	return nil, nil
}

func (_ *disabledStorage) FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	return nil, nil
}

func (_ *disabledStorage) InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	return nil
}

func (_ *disabledStorage) InsertBlockListItems(items []model.BlockListItem) error {
	return nil
}

func (_ *disabledStorage) DeleteBlockListItems(items []model.BlockListItem) error {
	return nil
}

func (_ *disabledStorage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	return nil, nil
}

func (_ *disabledStorage) Close() error {
	return nil
}
