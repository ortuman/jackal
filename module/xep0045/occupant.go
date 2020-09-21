/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

func (s *Muc) createOwner(ctx context.Context, userJID, occJID *jid.JID) (*mucmodel.Occupant, error) {
	o, err := s.newOccupant(ctx, userJID, occJID)
	if err != nil {
		return nil, err
	}
	o.SetAffiliation("owner")
	o.SetRole("moderator")
	err = s.repo.Occupant().UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Muc) newOccupant(ctx context.Context, userJID, occJID *jid.JID) (*mucmodel.Occupant, error) {
	// check if the occupant already exists
	o, err := s.repo.Occupant().FetchOccupant(ctx, occJID)
	if err != nil {
		return nil, err
	}
	if o != nil && userJID.ToBareJID().String() != o.BareJID.String() {
		return nil, fmt.Errorf("xep0045_occupant: User cannot use another user's occupant JID")
	}

	// if the occupant does not exist, create it
	if o == nil {
		o = &mucmodel.Occupant{
			OccupantJID: occJID,
			BareJID:     userJID.ToBareJID(),
			Resources:   make(map[string]bool),
		}
		err := s.repo.Occupant().UpsertOccupant(ctx, o)
		if err != nil {
			return nil, err
		}
	}

	return o, nil
}
