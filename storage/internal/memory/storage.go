/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memory

import (
	"errors"
	"sync"
)

// ErrMocked will be returned by when mocked error is activated.
var errMocked = errors.New("memstorage: mocked error")

// memoryStorage represents an in memory base storage instance.
type memoryStorage struct {
	b map[string][]byte

	mu          sync.RWMutex
	mockingErr  bool
	invokeLimit int32
	invokeCount int32
}

// newStorage returns a new in memory storage instance.
func newStorage() *memoryStorage {
	return &memoryStorage{b: make(map[string][]byte)}
}

// EnableMockedError enables in memory mocked error.
func (m *memoryStorage) EnableMockedError() {
	m.EnableMockedErrorWithInvokeLimit(1)
}

// EnableMockedErrorWithInvokeLimit enables in memory mocked error after a given invocation limit is reached.
func (m *memoryStorage) EnableMockedErrorWithInvokeLimit(invokeLimit int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockingErr = true
	m.invokeLimit = invokeLimit
	m.invokeCount = 0
}

// DisableMockedError disables in memory mocked error.
func (m *memoryStorage) DisableMockedError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockingErr = false
}

func (m *memoryStorage) inWriteLock(f func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.invokeCount++
	if m.invokeCount == m.invokeLimit {
		return errMocked
	}
	err := f()
	return err
}

func (m *memoryStorage) inReadLock(f func() error) error {
	m.mu.Lock()
	m.invokeCount++
	if m.invokeCount == m.invokeLimit {
		m.mu.Unlock()
		return errMocked
	}
	m.mu.Unlock()

	m.mu.RLock()
	err := f()
	m.mu.RUnlock()
	return err
}
