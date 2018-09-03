/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/storage/badgerdb"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/storage/sql"
	"github.com/ortuman/jackal/xmpp"
)

type userStorage interface {
	// InsertOrUpdateUser inserts a new user entity into storage,
	// or updates it in case it's been previously inserted.
	InsertOrUpdateUser(user *model.User) error

	// DeleteUser deletes a user entity from storage.
	DeleteUser(username string) error

	// FetchUser retrieves from storage a user entity.
	FetchUser(username string) (*model.User, error)

	// UserExists returns whether or not a user exists within storage.
	UserExists(username string) (bool, error)
}

type rosterStorage interface {
	// InsertOrUpdateRosterItem inserts a new roster item entity into storage,
	// or updates it in case it's been previously inserted.
	InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error)

	// DeleteRosterItem deletes a roster item entity from storage.
	DeleteRosterItem(username, jid string) (rostermodel.Version, error)

	// FetchRosterItems retrieves from storage all roster item entities
	// associated to a given user.
	FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error)

	// FetchRosterItem retrieves from storage a roster item entity.
	FetchRosterItem(username, jid string) (*rostermodel.Item, error)

	// InsertOrUpdateRosterNotification inserts a new roster notification entity
	// into storage, or updates it in case it's been previously inserted.
	InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error

	// DeleteRosterNotification deletes a roster notification entity from storage.
	DeleteRosterNotification(contact, jid string) error

	// FetchRosterNotification retrieves from storage a roster notification entity.
	FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error)

	// FetchRosterNotifications retrieves from storage all roster notifications
	// associated to a given user.
	FetchRosterNotifications(contact string) ([]rostermodel.Notification, error)
}

type offlineStorage interface {
	// InsertOfflineMessage inserts a new message element into
	// user's offline queue.
	InsertOfflineMessage(message *xmpp.Message, username string) error

	// CountOfflineMessages returns current length of user's offline queue.
	CountOfflineMessages(username string) (int, error)

	// FetchOfflineMessages retrieves from storage current user offline queue.
	FetchOfflineMessages(username string) ([]*xmpp.Message, error)

	// DeleteOfflineMessages clears a user offline queue.
	DeleteOfflineMessages(username string) error
}

type vCardStorage interface {
	// InsertOrUpdateVCard inserts a new vCard element into storage,
	// or updates it in case it's been previously inserted.
	InsertOrUpdateVCard(vCard xmpp.XElement, username string) error

	// FetchVCard retrieves from storage a vCard element associated
	// to a given user.
	FetchVCard(username string) (xmpp.XElement, error)
}

type privateStorage interface {
	// FetchPrivateXML retrieves from storage a private element.
	FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error)

	// InsertOrUpdatePrivateXML inserts a new private element into storage,
	// or updates it in case it's been previously inserted.
	InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error
}

type blockListStorage interface {
	// InsertBlockListItems inserts a set of block list item entities
	// into storage, only in case they haven't been previously inserted.
	InsertBlockListItems(items []model.BlockListItem) error

	// DeleteBlockListItems deletes a set of block list item entities from storage.
	DeleteBlockListItems(items []model.BlockListItem) error

	// FetchBlockListItems retrieves from storage all block list item entities
	// associated to a given user.
	FetchBlockListItems(username string) ([]model.BlockListItem, error)
}

// Storage represents an entity storage interface.
type Storage interface {
	userStorage
	offlineStorage
	rosterStorage
	vCardStorage
	privateStorage
	blockListStorage

	// Shutdown shuts down storage sub system.
	Shutdown()
}

var (
	instMu      sync.RWMutex
	inst        Storage
	initialized bool
)

// Initialize initializes storage sub system.
func Initialize(cfg *Config) {
	instMu.Lock()
	defer instMu.Unlock()
	if initialized {
		return
	}
	switch cfg.Type {
	case BadgerDB:
		inst = badgerdb.New(cfg.BadgerDB)
	case MySQL:
		inst = sql.New(cfg.MySQL)
	case Memory:
		inst = memstorage.New()
	default:
		// should not be reached
		break
	}
	initialized = true
}

// Instance returns global storage sub system.
func Instance() Storage {
	instMu.RLock()
	defer instMu.RUnlock()
	if inst == nil {
		log.Fatalf("storage subsystem not initialized")
	}
	return inst
}

// Shutdown shuts down storage sub system.
// This method should be used only for testing purposes.
func Shutdown() {
	instMu.Lock()
	defer instMu.Unlock()
	inst.Shutdown()
	inst = nil
	initialized = false
}

// ActivateMockedError forces the return of ErrMockedError from current storage manager.
// This method should only be used for testing purposes.
func ActivateMockedError() {
	instMu.Lock()
	defer instMu.Unlock()

	switch inst := inst.(type) {
	case *memstorage.Storage:
		inst.ActivateMockedError()
	}
}

// DeactivateMockedError disables mocked storage error from a previous activation.
// This method should only be used for testing purposes.
func DeactivateMockedError() {
	instMu.Lock()
	defer instMu.Unlock()

	switch inst := inst.(type) {
	case *memstorage.Storage:
		inst.DeactivateMockedError()
	}
}
