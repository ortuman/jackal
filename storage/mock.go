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

func (m *mockStorage) FetchUser(username string) (*model.User, error) {
	var ret *model.User
	err := m.inReadLock(func() error {
		ret = m.users[username]
		return nil
	})
	return ret, err
}

func (m *mockStorage) InsertOrUpdateUser(user *model.User) error {
	return m.inWriteLock(func() error {
		m.users[user.Username] = user
		return nil
	})
}

func (m *mockStorage) DeleteUser(username string) error {
	return m.inWriteLock(func() error {
		delete(m.users, username)
		return nil
	})
}

func (m *mockStorage) UserExists(username string) (bool, error) {
	var ret bool
	err := m.inReadLock(func() error {
		ret = m.users[username] != nil
		return nil
	})
	return ret, err
}

func (m *mockStorage) FetchRosterItems(user string) ([]model.RosterItem, model.RosterVersion, error) {
	var ris []model.RosterItem
	var v model.RosterVersion
	err := m.inReadLock(func() error {
		ris = m.rosterItems[user]
		v = m.rosterVersions[user]
		return nil
	})
	return ris, v, err
}

func (m *mockStorage) FetchRosterItem(user, contact string) (*model.RosterItem, error) {
	var ret *model.RosterItem
	err := m.inReadLock(func() error {
		ris := m.rosterItems[user]
		for _, ri := range ris {
			if ri.JID == contact {
				ret = &ri
				return nil
			}
		}
		return nil
	})
	return ret, err
}

func (m *mockStorage) InsertOrUpdateRosterItem(ri *model.RosterItem) (model.RosterVersion, error) {
	var v model.RosterVersion
	err := m.inWriteLock(func() error {
		ris := m.rosterItems[ri.Username]
		if ris != nil {
			for i, r := range ris {
				if r.JID == ri.JID {
					ris[i] = *ri
					goto done
				}
			}
			ris = append(ris, *ri)
		} else {
			ris = []model.RosterItem{*ri}
		}

	done:
		ver := m.rosterVersions[ri.Username]
		ver.Ver++
		m.rosterVersions[ri.Username] = ver
		ris[len(ris)-1].Ver = ver.Ver
		m.rosterItems[ri.Username] = ris
		return nil
	})
	return v, err
}

func (m *mockStorage) DeleteRosterItem(user, contact string) (model.RosterVersion, error) {
	var v model.RosterVersion
	err := m.inWriteLock(func() error {
		ris := m.rosterItems[user]
		for i, ri := range ris {
			if ri.JID == contact {
				m.rosterItems[user] = append(ris[:i], ris[i+1:]...)
				goto done
			}
		}
	done:
		v = m.rosterVersions[user]
		v.Ver++
		v.DeletionVer = v.Ver
		m.rosterVersions[user] = v
		return nil
	})
	return v, err
}

func (m *mockStorage) FetchRosterNotifications(contact string) ([]model.RosterNotification, error) {
	var ret []model.RosterNotification
	err := m.inReadLock(func() error {
		ret = m.rosterNotifications[contact]
		return nil
	})
	return ret, err
}

func (m *mockStorage) InsertOrUpdateRosterNotification(rn *model.RosterNotification) error {
	return m.inWriteLock(func() error {
		rns := m.rosterNotifications[rn.Contact]
		if rns != nil {
			for i, r := range rns {
				if r.JID == rn.JID {
					rns[i] = *rn
					goto done
				}
			}
			rns = append(rns, *rn)
		} else {
			rns = []model.RosterNotification{*rn}
		}
	done:
		m.rosterNotifications[rn.Contact] = rns
		return nil
	})
}

func (m *mockStorage) DeleteRosterNotification(contact, jid string) error {
	return m.inWriteLock(func() error {
		rns := m.rosterNotifications[contact]
		for i, rn := range rns {
			if rn.JID == jid {
				m.rosterNotifications[contact] = append(rns[:i], rns[i+1:]...)
				return nil
			}
		}
		return nil
	})
}

func (m *mockStorage) InsertOrUpdateVCard(vCard xml.XElement, username string) error {
	return m.inWriteLock(func() error {
		m.vCards[username] = xml.NewElementFromElement(vCard)
		return nil
	})
}

func (m *mockStorage) FetchVCard(username string) (xml.XElement, error) {
	var ret xml.XElement
	err := m.inReadLock(func() error {
		ret = m.vCards[username]
		return nil
	})
	return ret, err
}

func (m *mockStorage) InsertOrUpdatePrivateXML(privateXML []xml.XElement, namespace string, username string) error {
	return m.inWriteLock(func() error {
		var elems []xml.XElement
		for _, prv := range privateXML {
			elems = append(elems, xml.NewElementFromElement(prv))
		}
		m.privateXML[username+":"+namespace] = elems
		return nil
	})
}

func (m *mockStorage) FetchPrivateXML(namespace string, username string) ([]xml.XElement, error) {
	var ret []xml.XElement
	err := m.inReadLock(func() error {
		ret = m.privateXML[username+":"+namespace]
		return nil
	})
	return ret, err
}

func (m *mockStorage) InsertOfflineMessage(message xml.XElement, username string) error {
	return m.inWriteLock(func() error {
		msgs := m.offlineMessages[username]
		msgs = append(msgs, xml.NewElementFromElement(message))
		m.offlineMessages[username] = msgs
		return nil
	})
}

func (m *mockStorage) CountOfflineMessages(username string) (int, error) {
	var ret int
	err := m.inReadLock(func() error {
		ret = len(m.offlineMessages[username])
		return nil
	})
	return ret, err
}

func (m *mockStorage) FetchOfflineMessages(username string) ([]xml.XElement, error) {
	var ret []xml.XElement
	err := m.inReadLock(func() error {
		ret = m.offlineMessages[username]
		return nil
	})
	return ret, err
}

func (m *mockStorage) DeleteOfflineMessages(username string) error {
	return m.inWriteLock(func() error {
		delete(m.offlineMessages, username)
		return nil
	})
}

func (m *mockStorage) InsertOrUpdateBlockListItems(items []model.BlockListItem) error {
	return m.inWriteLock(func() error {
		for _, item := range items {
			bl := m.blockListItems[item.Username]
			if bl != nil {
				for _, blItem := range bl {
					if blItem.JID == item.JID {
						goto done
					}
				}
				m.blockListItems[item.Username] = append(bl, item)
			} else {
				m.blockListItems[item.Username] = []model.BlockListItem{item}
			}
		done:
		}
		return nil
	})
}

func (m *mockStorage) DeleteBlockListItems(items []model.BlockListItem) error {
	return m.inWriteLock(func() error {
		for _, itm := range items {
			bl := m.blockListItems[itm.Username]
			for i, blItem := range bl {
				if blItem.JID == itm.JID {
					m.blockListItems[itm.Username] = append(bl[:i], bl[i+1:]...)
					break
				}
			}
		}
		return nil
	})
}

func (m *mockStorage) DeleteBlockList(username string) error {
	return m.inWriteLock(func() error {
		delete(m.blockListItems, username)
		return nil
	})
}

func (m *mockStorage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	var ret []model.BlockListItem
	err := m.inReadLock(func() error {
		ret = m.blockListItems[username]
		return nil
	})
	return ret, err
}

func (m *mockStorage) activateMockedError() {
	atomic.StoreUint32(&m.mockErr, 1)
}

func (m *mockStorage) deactivateMockedError() {
	atomic.StoreUint32(&m.mockErr, 0)
}

func (m *mockStorage) inWriteLock(f func() error) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	err := f()
	m.mu.Unlock()
	return err
}

func (m *mockStorage) inReadLock(f func() error) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.RLock()
	err := f()
	m.mu.RUnlock()
	return err
}
