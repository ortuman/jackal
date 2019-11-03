package memstorage

import (
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/serializer"
)

func (m *Storage) InsertCapabilities(node, ver string, caps *model.Capabilities) error {
	b, err := serializer.Serialize(caps)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.bytes[capabilitiesKey(node, ver)] = b
		return nil
	})
}

func (m *Storage) HasCapabilities(node, ver string) (bool, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[capabilitiesKey(node, ver)]
		return nil
	}); err != nil {
		return false, err
	}
	return b != nil, nil
}

func (m *Storage) FetchCapabilities(node, ver string) (*model.Capabilities, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[capabilitiesKey(node, ver)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var caps model.Capabilities
	if err := serializer.Deserialize(b, &caps); err != nil {
		return nil, err
	}
	return &caps, nil
}

func capabilitiesKey(node, ver string) string {
	return "capabilities:" + node + ":" + ver
}
