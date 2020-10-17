/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/ortuman/jackal/xmpp"
)

func (s *Muc) createOwner(ctx context.Context, userJID, occJID *jid.JID) (*mucmodel.Occupant, error) {
	o, err := s.newOccupant(ctx, userJID, occJID)
	if err != nil {
		return nil, err
	}
	o.SetAffiliation("owner")
	err = s.repOccupant.UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Muc) newOccupant(ctx context.Context, userJID, occJID *jid.JID) (*mucmodel.Occupant, error) {
	// check if the occupant already exists
	o, err := s.repOccupant.FetchOccupant(ctx, occJID)
	if err != nil {
		return nil, err
	}

	if o != nil && userJID.ToBareJID().String() != o.BareJID.String() {
		return nil, fmt.Errorf("xep0045_occupant: User cannot use another user's occupant nick")
	}

	if o == nil {
		o, err = mucmodel.NewOccupant(occJID, userJID.ToBareJID())
		if err != nil {
			return nil, err
		}
	}
	o.AddResource(userJID.Resource())

	err = s.repOccupant.UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Muc) getOwnerFromIQ(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) (*mucmodel.Occupant, bool) {
	fromJID, err := jid.NewWithString(iq.From(), true)
	if err != nil {
		_ = s.router.Route(ctx, iq.BadRequestError())
		return nil, false
	}

	occJID, ok := room.UserToOccupant[*fromJID.ToBareJID()]
	if !ok {
		_ = s.router.Route(ctx, iq.ForbiddenError())
		return nil, false
	}

	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
		return nil, false
	}

	if !occ.IsOwner() {
		_ = s.router.Route(ctx, iq.ForbiddenError())
		return nil, false
	}

	return occ, true
}
