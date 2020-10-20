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
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
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

	if !userJID.IsFull() {
		return nil, fmt.Errorf("xep0045_occupant: User jid has to specify the resource")

	}
	o.AddResource(userJID.Resource())

	err = s.repOccupant.UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Muc) getOwnerFromIQ(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) (*mucmodel.Occupant, xmpp.Stanza) {
	fromJID, err := jid.NewWithString(iq.From(), true)
	if err != nil {
		return nil, iq.BadRequestError()
	}

	occJID, ok := room.GetOccupantJID(fromJID.ToBareJID())
	if !ok {
		return nil, iq.ForbiddenError()
	}

	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		return nil, iq.InternalServerError()
	}

	if !occ.IsOwner() {
		return nil, iq.ForbiddenError()
	}

	return occ, nil
}

func (s *Muc) getOccupantFromMessage(ctx context.Context, room *mucmodel.Room,
	message *xmpp.Message) (*mucmodel.Occupant, xmpp.Stanza) {
	occJID, ok := room.GetOccupantJID(message.FromJID().ToBareJID())
	if !ok {
		return nil, message.ForbiddenError()
	}

	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		return nil, message.InternalServerError()
	}
	return occ, nil
}

func (s *Muc) getOccupantFromPresence(ctx context.Context, room *mucmodel.Room,
	presence *xmpp.Presence) (*mucmodel.Occupant, xmpp.Stanza) {
	occJID, ok := room.GetOccupantJID(presence.FromJID().ToBareJID())
	if !ok {
		return nil, presence.ForbiddenError()
	}

	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		return nil, presence.InternalServerError()
	}
	return occ, nil
}
