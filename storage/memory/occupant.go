/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

// Room represents an in-memory room storage.
type Occupant struct {
	*memoryStorage
}

// NewOccupant returns an instance of Occupant in-memory storage.
func NewOccupant() *Occupant {
	return &Occupant{memoryStorage: newStorage()}
}

// UpsertOccupant inserts a new occupant entity into storage, or updates it in case it's been previously inserted.
func (m *Occupant) UpsertOccupant(_ context.Context, occ *mucmodel.Occupant) error {
	return m.saveEntity(occKey(occ.OccupantJID), occ)
}

// DeleteOccupant deletes an occupant entity from storage.
func (m *Occupant) DeleteOccupant(_ context.Context, occJID *jid.JID) error {
	return m.deleteKey(occKey(occJID))
}

// FetchOccupant retrieves from storage an occupant entity.
func (m *Occupant) FetchOccupant(_ context.Context, occJID *jid.JID) (*mucmodel.Occupant, error) {
	var occ mucmodel.Occupant
	ok, err := m.getEntity(occKey(occJID), &occ)
	switch err {
	case nil:
		if ok {
			return &occ, nil
		}
		return nil, nil
	default:
		return nil, err
	}
}

// OccupantExists returns whether or not an occupant exists within storage.
func (m *Occupant) OccupantExists(_ context.Context, occJID *jid.JID) (bool, error) {
	return m.keyExists(occKey(occJID))
}

func occKey(occJID *jid.JID) string {
	return "jid" + occJID.String()
}
