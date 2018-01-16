/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"sync"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/xml"
)

type User struct {
	Username string
	Password string
}

type RosterItem struct {
	User         string
	Contact      string
	Domain       string
	Name         string
	Subscription string
	Ask          bool
	Groups       []string
}

type storage interface {
	// User
	FetchUser(username string) (*User, error)

	InsertOrUpdateUser(user *User) error
	DeleteUser(username string) error

	UserExists(username string) (bool, error)

	// Roster
	InsertOrUpdateRosterItem(ri *RosterItem) error
	DeleteRosterItem(username, contact, domain string) error

	FetchRosterItem(username, contact, domain string) (*RosterItem, error)
	FetchRosterItems(username string) ([]RosterItem, error)

	// Roster approval notifications
	InsertOrUpdateRosterNotification(username, jid string, notification xml.Element) error
	DeleteRosterNotification(username, jid string) error

	FetchRosterNotifications(jid string) ([]xml.Element, error)

	// vCard
	FetchVCard(username string) (xml.Element, error)
	InsertOrUpdateVCard(vCard xml.Element, username string) error

	// Private XML
	FetchPrivateXML(namespace string, username string) ([]xml.Element, error)
	InsertOrUpdatePrivateXML(privateXML []xml.Element, namespace string, username string) error

	// Offline messages
	InsertOfflineMessage(message xml.Element, username string) error
	CountOfflineMessages(username string) (int, error)
	FetchOfflineMessages(username string) ([]xml.Element, error)
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
