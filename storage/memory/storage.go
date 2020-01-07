/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"errors"
	"sync"

	"github.com/ortuman/jackal/model/serializer"
)

var (
	mockErrMu   sync.RWMutex
	mockErr     bool
	invokeLimit int32
	invokeCount int32
)

// ErrMocked represents in memory mocked error value.
var ErrMocked = errors.New("memstorage: mocked error")

type memoryStorage struct {
	mu sync.RWMutex
	b  map[string][]byte
}

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

func (m *memoryStorage) saveEntity(k string, entity serializer.Serializer) error {
	b, err := serializer.Serialize(entity)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.b[k] = b
		return nil
	})
}

func (m *memoryStorage) saveEntities(k string, entities interface{}) error {
	b, err := serializer.SerializeSlice(entities)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.b[k] = b
		return nil
	})
}

func (m *memoryStorage) getEntity(k string, entity serializer.Deserializer) (bool, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[k]
		return nil
	}); err != nil {
		return false, err
	}
	if b == nil {
		return false, nil
	}
	if err := serializer.Deserialize(b, entity); err != nil {
		return false, err
	}
	return true, nil
}

func (m *memoryStorage) updateInWriteLock(k string, f func(b []byte) ([]byte, error)) error {
	return m.inWriteLock(func() error {
		b, err := f(m.b[k])
		if err != nil {
			return err
		}
		m.b[k] = b
		return nil
	})
}

func (m *memoryStorage) getEntities(k string, entities interface{}) (bool, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[k]
		return nil
	}); err != nil {
		return false, err
	}
	if b == nil {
		return false, nil
	}
	if err := serializer.DeserializeSlice(b, entities); err != nil {
		return false, err
	}
	return true, nil
}

func (m *memoryStorage) deleteKey(k string) error {
	return m.inWriteLock(func() error {
		delete(m.b, k)
		return nil
	})
}

func (m *memoryStorage) keyExists(k string) (bool, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[k]
		return nil
	}); err != nil {
		return false, err
	}
	return b != nil, nil
}

func checkMockedError() error {
	mockErrMu.Lock()
	defer mockErrMu.Unlock()

	if mockErr {
		invokeCount++
		if invokeCount >= invokeLimit {
			return ErrMocked
		}
	}
	return nil
}
