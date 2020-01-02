/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/model/serializer"
	"github.com/ortuman/jackal/xmpp"
)

type VCard struct {
	*memoryStorage
}

func NewVCard() *VCard {
	return &VCard{memoryStorage: newStorage()}
}

// UpsertVCard inserts a new vCard element into storage, or updates it in case it's been previously inserted.
func (m *VCard) UpsertVCard(_ context.Context, vCard xmpp.XElement, username string) error {
	b, err := serializer.Serialize(vCard)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.b[vCardKey(username)] = b
		return nil
	})
}

// FetchVCard retrieves from storage a vCard element associated to a given user.
func (m *VCard) FetchVCard(_ context.Context, username string) (xmpp.XElement, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[vCardKey(username)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var vCard xmpp.Element
	if err := serializer.Deserialize(b, &vCard); err != nil {
		return nil, err
	}
	return &vCard, nil
}

func vCardKey(username string) string {
	return "vCards:" + username
}
