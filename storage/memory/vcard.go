/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

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
	return m.saveEntity(vCardKey(username), vCard)
}

// FetchVCard retrieves from storage a vCard element associated to a given user.
func (m *VCard) FetchVCard(_ context.Context, username string) (xmpp.XElement, error) {
	var vCard xmpp.Element
	ok, err := m.getEntity(vCardKey(username), &vCard)
	switch err {
	case nil:
		if ok {
			return &vCard, nil
		}
		return nil, nil
	default:
		return nil, err
	}
}

func vCardKey(username string) string {
	return "vCards:" + username
}
