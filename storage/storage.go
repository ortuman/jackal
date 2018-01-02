/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package storage

import (
	"sync"

	"github.com/ortuman/jackal/storage/entity"
	"github.com/ortuman/jackal/xml"

	"github.com/ortuman/jackal/config"
)

type storage interface {
	// User
	FetchUser(username string) (*entity.User, error)

	InsertOrUpdateUser(user *entity.User) error
	DeleteUser(username string) error

	UserExists(username string) (bool, error)

	// Roster
	InsertOrUpdateRosterItem(username string, ri *entity.RosterItem) error
	DeleteRosterItem(username, jid string) error
	FetchRosterItems(username string) ([]entity.RosterItem, error)

	// vCard
	FetchVCard(username string) (*xml.Element, error)
	InsertOrUpdateVCard(vCard *xml.Element, username string) error

	// Private XML
	FetchPrivateXML(namespace string, username string) ([]*xml.Element, error)
	InsertOrUpdatePrivateXML(privateXML []*xml.Element, namespace string, username string) error

	// Offline messages
	InsertOfflineMessage(message *xml.Element, username string) error
	CountOfflineMessages(username string) (int, error)
	FetchOfflineMessages(username string) ([]*xml.Element, error)
	DeleteOfflineMessages(username string) error
}

// singleton interface
var (
	instance storage
	once     sync.Once
)

func Instance() storage {
	once.Do(func() {
		switch config.DefaultConfig.Storage.Type {
		case config.MySQL:
			instance = newMySQLStorage()
		default:
			// should not be reached
			break
		}
	})
	return instance
}
