/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"errors"
	"sync"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/xmpp"
)

// ErrMockedError will be returned by any Storage method
// when mocked error is activated.
var ErrMockedError = errors.New("memstorage: mocked error")

// Storage represents an in memory storage sub system.
type Storage struct {
	mockErrMu           sync.Mutex
	mockingErr          bool
	invokeLimit         int32
	invokeCount         int32
	mu                  sync.RWMutex
	users               map[string]*model.User
	rosterItems         map[string][]rostermodel.Item
	rosterVersions      map[string]rostermodel.Version
	rosterNotifications map[string][]rostermodel.Notification
	vCards              map[string]xmpp.XElement
	privateXML          map[string][]xmpp.XElement
	offlineMessages     map[string][]xmpp.Message
	blockListItems      map[string][]model.BlockListItem

	bytes map[string][]byte
}

// New returns a new in memory storage instance.
func New() *Storage {
	return &Storage{
		users:               make(map[string]*model.User),
		rosterItems:         make(map[string][]rostermodel.Item),
		rosterVersions:      make(map[string]rostermodel.Version),
		rosterNotifications: make(map[string][]rostermodel.Notification),
		vCards:              make(map[string]xmpp.XElement),
		privateXML:          make(map[string][]xmpp.XElement),
		offlineMessages:     make(map[string][]xmpp.Message),
		blockListItems:      make(map[string][]model.BlockListItem),
		bytes:               make(map[string][]byte),
	}
}

// IsClusterCompatible returns whether or not the underlying storage subsystem can be used in cluster mode.
func (m *Storage) IsClusterCompatible() bool { return false }

// Close shuts down in memory storage sub system.
func (m *Storage) Close() error {
	return nil
}

// EnableMockedError enables in memory mocked error.
func (m *Storage) EnableMockedError() {
	m.EnableMockedErrorWithInvokeLimit(1)
}

// EnableMockedErrorWithInvokeLimit enables in memory mocked error after a given invocation limit is reached.
func (m *Storage) EnableMockedErrorWithInvokeLimit(invokeLimit int32) {
	m.mockErrMu.Lock()
	defer m.mockErrMu.Unlock()
	m.mockingErr = true
	m.invokeLimit = invokeLimit
	m.invokeCount = 0
}

// DisableMockedError disables in memory mocked error.
func (m *Storage) DisableMockedError() {
	m.mockErrMu.Lock()
	defer m.mockErrMu.Unlock()
	m.mockingErr = false
}

func (m *Storage) inWriteLock(f func() error) error {
	m.mockErrMu.Lock()
	defer m.mockErrMu.Unlock()
	m.invokeCount++
	if m.invokeCount == m.invokeLimit {
		return ErrMockedError
	}
	m.mu.Lock()
	err := f()
	m.mu.Unlock()
	return err
}

func (m *Storage) inReadLock(f func() error) error {
	m.mockErrMu.Lock()
	defer m.mockErrMu.Unlock()
	m.invokeCount++
	if m.invokeCount == m.invokeLimit {
		return ErrMockedError
	}
	m.mu.RLock()
	err := f()
	m.mu.RUnlock()
	return err
}
