/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"errors"
	"sync"
)

var (
	mockErrMu   sync.RWMutex
	mockErr     bool
	invokeLimit int32
	invokeCount int32
)

// ErrMocked will be returned by when mocked error is activated.
var errMocked = errors.New("memstorage: mocked error")

// memoryStorage represents an in memory base storage instance.
type memoryStorage struct {
	mu sync.RWMutex
	b  map[string][]byte
}

// newStorage returns a new in memory storage instance.
func newStorage() *memoryStorage {
	return &memoryStorage{b: make(map[string][]byte)}
}

// EnableMockedError enables in memory mocked error.
func EnableMockedError() {
	EnableMockedErrorWithInvokeLimit(1)
}

// EnableMockedErrorWithInvokeLimit enables in memory mocked error after a given invocation limit is reached.
func EnableMockedErrorWithInvokeLimit(limit int32) {
	mockErrMu.Lock()
	defer mockErrMu.Unlock()
	mockErr = true
	invokeLimit = limit
	invokeCount = 0
}

// DisableMockedError disables in memory mocked error.
func DisableMockedError() {
	mockErrMu.Lock()
	defer mockErrMu.Unlock()
	mockErr = false
}

func (m *memoryStorage) inWriteLock(f func() error) error {
	if err := checkMockedError(); err != nil {
		return err
	}
	m.mu.Lock()
	err := f()
	m.mu.Unlock()
	return err
}

func (m *memoryStorage) inReadLock(f func() error) error {
	if err := checkMockedError(); err != nil {
		return err
	}
	m.mu.RLock()
	err := f()
	m.mu.RUnlock()
	return err
}

func checkMockedError() error {
	mockErrMu.Lock()
	defer mockErrMu.Unlock()

	if mockErr {
		invokeCount++
		if invokeCount >= invokeLimit {
			return errMocked
		}
	}
	return nil
}
