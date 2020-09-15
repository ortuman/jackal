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

func (s *Muc) createOwner(ctx context.Context, occJID *jid.JID, nick string, fullJID *jid.JID) (*mucmodel.Occupant, error) {
	o := &mucmodel.Occupant{
		OccupantJID: occJID,
		Nick:        nick,
		BareJID:     fullJID.ToBareJID(),
	}
	o.SetAffiliation("owner")
	o.SetRole("moderator")
	err := s.reps.Occupant().UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}
