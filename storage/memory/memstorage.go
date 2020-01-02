/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"errors"
	"sync"
)

// ErrMockedError will be returned by any Storage method when mocked error is activated.
var ErrMockedError = errors.New("memstorage: mocked error")

// Storage represents an in memory storage sub system.
type Storage struct {
	mu    sync.RWMutex
	bytes map[string][]byte
}

// New returns a new in memory storage instance.
func New2() *Storage {
	return &Storage{
		bytes: make(map[string][]byte),
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
	EnableMockedErrorWithInvokeLimit(1)
}

// EnableMockedErrorWithInvokeLimit enables in memory mocked error after a given invocation limit is reached.
func (m *Storage) EnableMockedErrorWithInvokeLimit(limit int32) {
	EnableMockedErrorWithInvokeLimit(limit)
}

// DisableMockedError disables in memory mocked error.
func (m *Storage) DisableMockedError() {
	DisableMockedError()
}

func (m *Storage) inWriteLock(f func() error) error {
	if err := checkMockedError(); err != nil {
		return err
	}
	m.mu.Lock()
	err := f()
	m.mu.Unlock()
	return err
}

func (m *Storage) inReadLock(f func() error) error {
	if err := checkMockedError(); err != nil {
		return err
	}
	m.mu.RLock()
	err := f()
	m.mu.RUnlock()
	return err
}
