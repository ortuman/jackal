/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

// User defines user repository operations
type Occupant interface {
	// UpsertOccupant inserts a new occupant entity into storage, or updates it if previously inserted.
	UpsertOccupant(ctx context.Context, occ *mucmodel.Occupant) error

	// DeleteOccupant deletes a occupant entity from storage.
	DeleteOccupant(ctx context.Context, occJID *jid.JID) error

	// FetchOccupant retrieves an occupant entity from storage.
	FetchOccupant(ctx context.Context, occJID *jid.JID) (*mucmodel.Occupant, error)

	// OccupantExists tells whether or not an occupant exists within storage.
	OccupantExists(ctx context.Context, occJID *jid.JID) (bool, error)
}
