/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"fmt"
	"sync"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/storage/badgerdb"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/storage/sql"
	"github.com/ortuman/jackal/xmpp"
)

// IsClusterCompatible returns whether or not the underlying storage subsystem can be used in cluster mode.
func IsClusterCompatible() bool {
	return false
}

type userStorage interface {
	InsertOrUpdateUser(user *model.User) error
	DeleteUser(username string) error
	FetchUser(username string) (*model.User, error)
	UserExists(username string) (bool, error)
}

// InsertOrUpdateUser inserts a new user entity into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdateUser(user *model.User) error {
	return instance().InsertOrUpdateUser(user)
}

// DeleteUser deletes a user entity from storage.
func DeleteUser(username string) error {
	return instance().DeleteUser(username)
}

// FetchUser retrieves from storage a user entity.
func FetchUser(username string) (*model.User, error) {
	return instance().FetchUser(username)
}

// UserExists returns whether or not a user exists within storage.
func UserExists(username string) (bool, error) {
	return instance().UserExists(username)
}

type rosterStorage interface {
	InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error)
	DeleteRosterItem(username, jid string) (rostermodel.Version, error)
	FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error)
	FetchRosterItem(username, jid string) (*rostermodel.Item, error)
	InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error
	DeleteRosterNotification(contact, jid string) error
	FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error)
	FetchRosterNotifications(contact string) ([]rostermodel.Notification, error)
}

// InsertOrUpdateRosterItem inserts a new roster item entity into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdateRosterItem(ri *rostermodel.Item) (rostermodel.Version, error) {
	return instance().InsertOrUpdateRosterItem(ri)
}

// DeleteRosterItem deletes a roster item entity from storage.
func DeleteRosterItem(username, jid string) (rostermodel.Version, error) {
	return instance().DeleteRosterItem(username, jid)
}

// FetchRosterItems retrieves from storage all roster item entities
// associated to a given user.
func FetchRosterItems(username string) ([]rostermodel.Item, rostermodel.Version, error) {
	return instance().FetchRosterItems(username)
}

// FetchRosterItem retrieves from storage a roster item entity.
func FetchRosterItem(username, jid string) (*rostermodel.Item, error) {
	return instance().FetchRosterItem(username, jid)
}

// InsertOrUpdateRosterNotification inserts a new roster notification entity
// into storage, or updates it in case it's been previously inserted.
func InsertOrUpdateRosterNotification(rn *rostermodel.Notification) error {
	return instance().InsertOrUpdateRosterNotification(rn)
}

// DeleteRosterNotification deletes a roster notification entity from storage.
func DeleteRosterNotification(contact, jid string) error {
	return instance().DeleteRosterNotification(contact, jid)
}

// FetchRosterNotification retrieves from storage a roster notification entity.
func FetchRosterNotification(contact string, jid string) (*rostermodel.Notification, error) {
	return instance().FetchRosterNotification(contact, jid)
}

// FetchRosterNotifications retrieves from storage all roster notifications
// associated to a given user.
func FetchRosterNotifications(contact string) ([]rostermodel.Notification, error) {
	return instance().FetchRosterNotifications(contact)
}

type offlineStorage interface {
	InsertOfflineMessage(message *xmpp.Message, username string) error
	CountOfflineMessages(username string) (int, error)
	FetchOfflineMessages(username string) ([]*xmpp.Message, error)
	DeleteOfflineMessages(username string) error
}

// InsertOfflineMessage inserts a new message element into
// user's offline queue.
func InsertOfflineMessage(message *xmpp.Message, username string) error {
	return instance().InsertOfflineMessage(message, username)
}

// CountOfflineMessages returns current length of user's offline queue.
func CountOfflineMessages(username string) (int, error) {
	return instance().CountOfflineMessages(username)
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func FetchOfflineMessages(username string) ([]*xmpp.Message, error) {
	return instance().FetchOfflineMessages(username)
}

// DeleteOfflineMessages clears a user offline queue.
func DeleteOfflineMessages(username string) error {
	return instance().DeleteOfflineMessages(username)
}

type vCardStorage interface {
	InsertOrUpdateVCard(vCard xmpp.XElement, username string) error
	FetchVCard(username string) (xmpp.XElement, error)
}

// InsertOrUpdateVCard inserts a new vCard element into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdateVCard(vCard xmpp.XElement, username string) error {
	return instance().InsertOrUpdateVCard(vCard, username)
}

// FetchVCard retrieves from storage a vCard element associated
// to a given user.
func FetchVCard(username string) (xmpp.XElement, error) {
	return instance().FetchVCard(username)
}

type privateStorage interface {
	FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error)
	InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error
}

// FetchPrivateXML retrieves from storage a private element.
func FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	return instance().FetchPrivateXML(namespace, username)
}

// InsertOrUpdatePrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	return instance().InsertOrUpdatePrivateXML(privateXML, namespace, username)
}

type blockListStorage interface {
	InsertBlockListItems(items []model.BlockListItem) error
	DeleteBlockListItems(items []model.BlockListItem) error
	FetchBlockListItems(username string) ([]model.BlockListItem, error)
}

// InsertBlockListItems inserts a set of block list item entities
// into storage, only in case they haven't been previously inserted.
func InsertBlockListItems(items []model.BlockListItem) error {
	return instance().InsertBlockListItems(items)
}

// DeleteBlockListItems deletes a set of block list item entities from storage.
func DeleteBlockListItems(items []model.BlockListItem) error {
	return instance().DeleteBlockListItems(items)
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	return instance().FetchBlockListItems(username)
}

// Storage represents an entity storage interface.
type Storage interface {
	Close() error

	IsClusterCompatible() bool

	userStorage
	offlineStorage
	rosterStorage
	vCardStorage
	privateStorage
	blockListStorage
}

var (
	instMu sync.RWMutex
	inst   Storage
)

var Disabled Storage = &disabledStorage{}

func init() {
	inst = Disabled
}

func Set(storage Storage) {
	instMu.Lock()
	_ = inst.Close()
	inst = storage
	instMu.Unlock()
}

func Unset() {
	Set(Disabled)
}

func instance() Storage {
	instMu.RLock()
	s := inst
	instMu.RUnlock()
	return s
}

// Initialize initializes storage sub system.
func New(config *Config) (Storage, error) {
	switch config.Type {
	case BadgerDB:
		return badgerdb.New(config.BadgerDB), nil
	case MySQL:
		return sql.New(config.MySQL), nil
	case Memory:
		return memstorage.New(), nil
	default:
		return nil, fmt.Errorf("storage: unrecognized storage type: %d", config.Type)
	}
}
