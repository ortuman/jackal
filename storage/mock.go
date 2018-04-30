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
	mockErr             uint32
	mu                  sync.RWMutex
	users               map[string]*model.User
	rosterItems         map[string][]model.RosterItem
	rosterVersions      map[string]model.RosterVersion
	rosterNotifications map[string][]model.RosterNotification
	vCards              map[string]xml.XElement
	privateXML          map[string][]xml.XElement
	offlineMessages     map[string][]xml.XElement
	blockListItems      map[string][]model.BlockListItem
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		users:               make(map[string]*model.User),
		rosterItems:         make(map[string][]model.RosterItem),
		rosterVersions:      make(map[string]model.RosterVersion),
		rosterNotifications: make(map[string][]model.RosterNotification),
		vCards:              make(map[string]xml.XElement),
		privateXML:          make(map[string][]xml.XElement),
		offlineMessages:     make(map[string][]xml.XElement),
		blockListItems:      make(map[string][]model.BlockListItem),
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *mockStorage) InsertOrUpdateUser(user *model.User) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.Username] = user
	return nil
}

func (m *mockStorage) DeleteUser(username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, username)
	return nil
}

func (m *mockStorage) UserExists(username string) (bool, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return false, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users[username] != nil, nil
}

func (m *mockStorage) FetchRosterItems(user string) ([]model.RosterItem, model.RosterVersion, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, model.RosterVersion{}, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rosterItems[user], m.rosterVersions[user], nil
}

func (m *mockStorage) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	rosterItems := m.rosterItems[user]
	for _, rosterItem := range rosterItems {
		if rosterItem.JID == contact {
			return &rosterItem, nil
		}
	}
	return nil, nil
}

func (m *mockStorage) InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return model.RosterVersion{}, ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	rosterItems := m.rosterItems[ri.Username]
	if rosterItems != nil {
		for i, rosterItem := range rosterItems {
			if rosterItem.JID == ri.JID {
				rosterItems[i] = *ri
				goto updateRosterItems
			}
		}
		rosterItems = append(rosterItems, *ri)
	} else {
		rosterItems = []model.RosterItem{*ri}
	}

updateRosterItems:
	ver := m.rosterVersions[ri.Username]
	ver.Ver++
	m.rosterVersions[ri.Username] = ver
	rosterItems[len(rosterItems)-1].Ver = ver.Ver
	m.rosterItems[ri.Username] = rosterItems
	return ver, nil
}

func (m *mockStorage) DeleteRosterItem(user, contact string) (model.RosterVersion, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return model.RosterVersion{}, ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	rosterItems := m.rosterItems[user]
	for i, rosterItem := range rosterItems {
		if rosterItem.JID == contact {
			m.rosterItems[user] = append(rosterItems[:i], rosterItems[i+1:]...)
			goto deletionDone
		}
	}

deletionDone:
	v := m.rosterVersions[user]
	v.Ver++
	v.DeletionVer = v.Ver
	m.rosterVersions[user] = v
	return v, nil
}

func (m *mockStorage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rosterNotifications[contact], nil
}

func (m *mockStorage) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	rosterNotifications := m.rosterNotifications[rn.Contact]
	if rosterNotifications != nil {
		for i, rosterNotification := range rosterNotifications {
			if rosterNotification.JID == rn.JID {
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

func (m *mockStorage) DeleteRosterNotification(contact, jid string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	rosterNotifications := m.rosterNotifications[contact]
	for i, rosterNotification := range rosterNotifications {
		if rosterNotification.JID == jid {
			m.rosterNotifications[contact] = append(rosterNotifications[:i], rosterNotifications[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockStorage) InsertOrUpdateVCard(vCard xml.XElement, username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vCards[username] = xml.NewElementFromElement(vCard)
	return nil
}

func (m *mockStorage) FetchVCard(username string) (xml.XElement, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vCards[username], nil
}

func (m *mockStorage) InsertOrUpdatePrivateXML(privateXML []xml.XElement, namespace string, username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// copy elements
	var prvXML []xml.XElement
	for _, prv := range privateXML {
		prvXML = append(prvXML, xml.NewElementFromElement(prv))
	}
	m.privateXML[username+":"+namespace] = prvXML
	return nil
}

func (m *mockStorage) FetchPrivateXML(namespace string, username string) ([]xml.XElement, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.privateXML[username+":"+namespace], nil
}

func (m *mockStorage) InsertOfflineMessage(message xml.XElement, username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	offlineMessages := m.offlineMessages[username]
	offlineMessages = append(offlineMessages, xml.NewElementFromElement(message))
	m.offlineMessages[username] = offlineMessages
	return nil
}

func (m *mockStorage) CountOfflineMessages(username string) (int, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return 0, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.offlineMessages[username]), nil
}

func (m *mockStorage) FetchOfflineMessages(username string) ([]xml.XElement, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.offlineMessages[username], nil
}

func (m *mockStorage) DeleteOfflineMessages(username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.offlineMessages, username)
	return nil
}

func (m *mockStorage) InsertOrUpdateBlockListItems(items []model.BlockListItem) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range items {
		bl := m.blockListItems[item.Username]
		if bl != nil {
			for _, blItem := range bl {
				if blItem.JID == item.JID {
					goto itemInserted
				}
			}
			m.blockListItems[item.Username] = append(bl, item)
		} else {
			m.blockListItems[item.Username] = []model.BlockListItem{item}
		}
	itemInserted:
	}
	return nil
}

func (m *mockStorage) DeleteBlockListItems(items []model.BlockListItem) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range items {
		bl := m.blockListItems[item.Username]
		for i, blItem := range bl {
			if blItem.JID == item.JID {
				m.blockListItems[item.Username] = append(bl[:i], bl[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *mockStorage) DeleteBlockList(username string) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.blockListItems, username)
	return nil
}

func (m *mockStorage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return nil, ErrMockedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.blockListItems[username], nil
}
