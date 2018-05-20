/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/badgerdb"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/storage/sql"
	"github.com/ortuman/jackal/xml"
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
	InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error)

	// DeleteRosterItem deletes a roster item entity from storage.
	DeleteRosterItem(username, jid string) (model.RosterVersion, error)

	// FetchRosterItems retrieves from storage all roster item entities
	// associated to a given user.
	FetchRosterItems(username string) ([]model.RosterItem, model.RosterVersion, error)

	// FetchRosterItem retrieves from storage a roster item entity.
	FetchRosterItem(username, jid string) (*model.RosterItem, error)

	// InsertOrUpdateRosterNotification inserts a new roster notification entity
	// into storage, or updates it in case it's been previously inserted.
	InsertOrUpdateRosterNotification(rn *model.RosterNotification) error

	// DeleteRosterNotification deletes a roster notification entity from storage.
	DeleteRosterNotification(contact, jid string) error

	// FetchRosterNotifications retrieves from storage all roster notifications
	// associated to a given user.
	FetchRosterNotifications(contact string) ([]model.RosterNotification, error)
}

type offlineStorage interface {
	// InsertOfflineMessage inserts a new message element into
	// user's offline queue.
	InsertOfflineMessage(message xml.XElement, username string) error

	// CountOfflineMessages returns current length of user's offline queue.
	CountOfflineMessages(username string) (int, error)

	// FetchOfflineMessages retrieves from storage current user offline queue.
	FetchOfflineMessages(username string) ([]xml.XElement, error)

	// DeleteOfflineMessages clears a user offline queue.
	DeleteOfflineMessages(username string) error
}

type vCardStorage interface {
	// InsertOrUpdateVCard inserts a new vCard element into storage,
	// or updates it in case it's been previously inserted.
	InsertOrUpdateVCard(vCard xml.XElement, username string) error

	// FetchVCard retrieves from storage a vCard element associated
	// to a given user.
	FetchVCard(username string) (xml.XElement, error)
}

type privateStorage interface {
	// FetchPrivateXML retrieves from storage a private element.
	FetchPrivateXML(namespace string, username string) ([]xml.XElement, error)

	// InsertOrUpdatePrivateXML inserts a new private element into storage,
	// or updates it in case it's been previously inserted.
	InsertOrUpdatePrivateXML(privateXML []xml.XElement, namespace string, username string) error
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
	inst        Storage
	instMu      sync.RWMutex
	initialized uint32
)

// Initialize initializes storage sub system.
func Initialize(cfg *Config) {
	if atomic.CompareAndSwapUint32(&initialized, 0, 1) {
		instMu.Lock()
		defer instMu.Unlock()

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
	}
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
	if atomic.CompareAndSwapUint32(&initialized, 1, 0) {
		instMu.Lock()
		defer instMu.Unlock()

		inst.Shutdown()
		inst = nil
	}
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
