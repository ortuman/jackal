/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memory

import (
	"errors"
	"sync"
)

// ErrMockedError will be returned by any Storage method when mocked error is activated.
var ErrMockedError = errors.New("memstorage: mocked error")

// Storage represents an in memory storage sub system.
type Storage struct {
	mockErrMu   sync.Mutex
	mockingErr  bool
	invokeLimit int32
	invokeCount int32
	mu          sync.RWMutex
	bytes       map[string][]byte
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
