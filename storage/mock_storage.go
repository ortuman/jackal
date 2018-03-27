/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
)

type mockStorage struct {
	mockErr               uint32
	usersMu               sync.RWMutex
	users                 map[string]*model.User
	rosterItemsMu         sync.RWMutex
	rosterItems           map[string][]model.RosterItem
	rosterNotificationsMu sync.RWMutex
	rosterNotifications   map[string][]model.RosterNotification
	vCardsMu              sync.RWMutex
	vCards                map[string]xml.ElementNode
	privateXMLMu          sync.RWMutex
	privateXML            map[string][]xml.ElementNode
	offlineMessagesMu     sync.RWMutex
	offlineMessages       map[string][]xml.ElementNode
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		users:               make(map[string]*model.User),
		rosterItems:         make(map[string][]model.RosterItem),
		rosterNotifications: make(map[string][]model.RosterNotification),
		vCards:              make(map[string]xml.ElementNode),
		privateXML:          make(map[string][]xml.ElementNode),
		offlineMessages:     make(map[string][]xml.ElementNode),
	}
}

func (m *mockStorage) Shutdown() {
}

func (m *mockStorage) activateMockedError() {
	atomic.StoreUint32(&m.mockErr, 1)
}

func (m *mockStorage) deactivateMockedError() {
	atomic.StoreUint32(&m.mockErr, 0)
}

func (m *mockStorage) FetchUser(username string) (*model.User, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.usersMu.RLock()
	defer m.usersMu.RUnlock()
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *mockStorage) InsertOrUpdateUser(user *model.User) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.usersMu.Lock()
	defer m.usersMu.Unlock()
	m.users[user.Username] = user
	return nil
}

func (m *mockStorage) DeleteUser(username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.usersMu.Lock()
	defer m.usersMu.Unlock()
	delete(m.users, username)
	return nil
}

func (m *mockStorage) UserExists(username string) (bool, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return false, ErrMockedError
	}
	m.usersMu.RLock()
	defer m.usersMu.RUnlock()
	return m.users[username] != nil, nil
}

func (m *mockStorage) FetchRosterItems(user string) ([]model.RosterItem, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.rosterItemsMu.RLock()
	defer m.rosterItemsMu.RUnlock()
	return m.rosterItems[user], nil
}

func (m *mockStorage) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.rosterItemsMu.RLock()
	defer m.rosterItemsMu.RUnlock()
	rosterItems := m.rosterItems[user]
	for _, rosterItem := range rosterItems {
		if rosterItem.Contact == contact {
			return &rosterItem, nil
		}
	}
	return nil, nil
}

func (m *mockStorage) InsertOrUpdateRosterItem(ri *model.RosterItem) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.rosterItemsMu.Lock()
	defer m.rosterItemsMu.Unlock()
	rosterItems := m.rosterItems[ri.User]
	if rosterItems != nil {
		for i, rosterItem := range rosterItems {
			if rosterItem.Contact == ri.Contact {
				rosterItems[i] = *ri
				goto updateRosterItems
			}
		}
		rosterItems = append(rosterItems, *ri)
	} else {
		rosterItems = []model.RosterItem{*ri}
	}
updateRosterItems:
	m.rosterItems[ri.User] = rosterItems
	return nil
}

func (m *mockStorage) DeleteRosterItem(user, contact string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.rosterItemsMu.Lock()
	defer m.rosterItemsMu.Unlock()
	rosterItems := m.rosterItems[user]
	for i, rosterItem := range rosterItems {
		if rosterItem.Contact == contact {
			m.rosterItems[user] = append(rosterItems[:i], rosterItems[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockStorage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.rosterItemsMu.RLock()
	defer m.rosterItemsMu.RUnlock()
	return m.rosterNotifications[contact], nil
}

func (m *mockStorage) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.rosterItemsMu.Lock()
	defer m.rosterItemsMu.Unlock()
	rosterNotifications := m.rosterNotifications[rn.Contact]
	if rosterNotifications != nil {
		for i, rosterNotification := range rosterNotifications {
			if rosterNotification.User == rn.User {
				rosterNotifications[i] = *rn
				goto updateRosterNotifications
			}
		}
		rosterNotifications = append(rosterNotifications, *rn)
	} else {
		rosterNotifications = []model.RosterNotification{*rn}
	}
updateRosterNotifications:
	m.rosterNotifications[rn.Contact] = rosterNotifications
	return nil
}

func (m *mockStorage) DeleteRosterNotification(user, contact string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.rosterItemsMu.Lock()
	defer m.rosterItemsMu.Unlock()
	rosterNotifications := m.rosterNotifications[contact]
	for i, rosterNotification := range rosterNotifications {
		if rosterNotification.User == user {
			m.rosterNotifications[contact] = append(rosterNotifications[:i], rosterNotifications[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockStorage) InsertOrUpdateVCard(vCard xml.ElementNode, username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.vCardsMu.Lock()
	defer m.vCardsMu.Unlock()

	m.vCards[username] = xml.NewElementFromElement(vCard)
	return nil
}

func (m *mockStorage) FetchVCard(username string) (xml.ElementNode, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.vCardsMu.RLock()
	defer m.vCardsMu.RUnlock()
	return m.vCards[username], nil
}

func (m *mockStorage) InsertOrUpdatePrivateXML(privateXML []xml.ElementNode, namespace string, username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.privateXMLMu.Lock()
	defer m.privateXMLMu.Unlock()

	// copy elements
	var prvXML []xml.ElementNode
	for _, prv := range privateXML {
		prvXML = append(prvXML, xml.NewElementFromElement(prv))
	}
	m.privateXML[username+":"+namespace] = prvXML
	return nil
}

func (m *mockStorage) FetchPrivateXML(namespace string, username string) ([]xml.ElementNode, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.privateXMLMu.RLock()
	defer m.privateXMLMu.RUnlock()
	return m.privateXML[username+":"+namespace], nil
}

func (m *mockStorage) InsertOfflineMessage(message xml.ElementNode, username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.offlineMessagesMu.Lock()
	defer m.offlineMessagesMu.Unlock()
	offlineMessages := m.offlineMessages[username]
	offlineMessages = append(offlineMessages, xml.NewElementFromElement(message))
	m.offlineMessages[username] = offlineMessages
	return nil
}

func (m *mockStorage) CountOfflineMessages(username string) (int, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return 0, ErrMockedError
	}
	m.offlineMessagesMu.RLock()
	defer m.offlineMessagesMu.RUnlock()
	return len(m.offlineMessages[username]), nil
}

func (m *mockStorage) FetchOfflineMessages(username string) ([]xml.ElementNode, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.offlineMessagesMu.RLock()
	defer m.offlineMessagesMu.RUnlock()
	return m.offlineMessages[username], nil
}

func (m *mockStorage) DeleteOfflineMessages(username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.offlineMessagesMu.Lock()
	defer m.offlineMessagesMu.Unlock()
	delete(m.offlineMessages, username)
	return nil
}
