/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func (s *Muc) messageEveryone(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	if !s.userHasVoice(ctx, room, message.FromJID(), message) {
		return
	}

	senderOccupantJID, _ := room.UserToOccupant[*message.FromJID().ToBareJID()]
	for _, occJID := range room.UserToOccupant {
		s.messageOccupant(ctx, &occJID, &senderOccupantJID, message)
	}
	return
}

func (s *Muc) userHasVoice(ctx context.Context, room *mucmodel.Room, userJID *jid.JID,
	message *xmpp.Message) bool {
	// user has to be occupant of the room
	occJID, found := room.UserToOccupant[*userJID.ToBareJID()]
	if !found {
		_ = s.router.Route(ctx, message.NotAcceptableError())
		return false
	}
	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, message.InternalServerError())
		return false
	}

	if room.Config.Moderated && (occ.IsVisitor() || occ.HasNoRole()) {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return false
	}

	return true
}

func (s *Muc) messageOccupant(ctx context.Context, occJID, senderJID *jid.JID,
	message *xmpp.Message) {
	occupant, err := s.repOccupant.FetchOccupant(ctx, occJID)
	if err != nil {
		log.Error(err)
		return
	}

	message.SetFromJID(senderJID)
	for resource, _ := range occupant.Resources {
		to := addResourceToBareJID(occupant.BareJID, resource)
		message.SetToJID(to)
		_ = s.router.Route(ctx, message)
	}
	return
}
