/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

const moderatorRole = "moderator"
const ownerAff = "owner"

func (s *Muc) createOwner(ctx context.Context, occJID *jid.JID, nick string, fullJID *jid.JID) (*mucmodel.Occupant, error) {
	o := &mucmodel.Occupant{
		OccupantJID: occJID,
		Nick:        nick,
		FullJID:     fullJID,
		Affiliation: ownerAff,
		Role:        moderatorRole,
	}
	err := s.reps.Occupant().UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}
