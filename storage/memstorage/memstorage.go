/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
)

// ErrMockedError will be returned by any Storage method
// when mocked error is activated.
var ErrMockedError = errors.New("storage mocked error")

type Storage struct {
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

func New() *Storage {
	return &Storage{
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

func (m *Storage) Shutdown() {
}

func (m *Storage) ActivateMockedError() {
	atomic.StoreUint32(&m.mockErr, 1)
}

func (m *Storage) DeactivateMockedError() {
	atomic.StoreUint32(&m.mockErr, 0)
}

func (m *Storage) inWriteLock(f func() error) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.Lock()
	err := f()
	m.mu.Unlock()
	return err
}

func (m *Storage) inReadLock(f func() error) error {
	if atomic.LoadUint32(&m.mockErr) == 1 {
		return ErrMockedError
	}
	m.mu.RLock()
	err := f()
	m.mu.RUnlock()
	return err
}
