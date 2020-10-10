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

func isDeclineInvitation(message *xmpp.Message) bool {
	x := message.Elements().Child("x")
	if x == nil || x.Namespace() != mucNamespaceUser {
		return false
	}
	decline := x.Elements().Child("decline")
	if decline == nil {
		return false
	}
	return true
}

func (s *Muc) declineInvitation(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	_, invited := room.InvitedUsers[*message.FromJID().ToBareJID()]
	if !invited {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return
	}

	// delete the user from the list of invited
	delete(room.InvitedUsers, *message.FromJID().ToBareJID())
	s.repRoom.UpsertRoom(ctx, room)

	s.forwardDeclineToUser(ctx, room, message)
}

func (s *Muc) forwardDeclineToUser(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	msg := getDeclineStanza(room, message)
	_ = s.router.Route(ctx, msg)
}

func isInvite(message *xmpp.Message) bool {
	x := message.Elements().Child("x")
	if x == nil || x.Namespace() != mucNamespaceUser {
		return false
	}
	invite := x.Elements().Child("invite")
	if invite == nil {
		return false
	}
	return true
}

func (s *Muc) inviteUser(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {

	if !s.userHasVoice(ctx, room, message.FromJID(), message) {
		return
	}

	occJID, _ := room.UserToOccupant[*message.FromJID().ToBareJID()]
	occ, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
	if !room.Config.AllowInvites || (!room.Config.Open && !occ.IsAdmin() && !occ.IsOwner()) {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return
	}

	// add to the list of invited users
	invJID := getInvitedUserJID(message)
	room.InvitedUsers[*invJID] = true
	s.repRoom.UpsertRoom(ctx, room)

	s.forwardInviteToUser(ctx, room, message)
}

func (s *Muc) forwardInviteToUser(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	inviteFrom := message.FromJID()
	inviteTo := getInvitedUserJID(message)

	msg := getInvitationStanza(room, inviteFrom, inviteTo, message)

	_ = s.router.Route(ctx, msg)
}

func getInvitedUserJID(message *xmpp.Message) *jid.JID {
	invJIDStr := message.Elements().Child("x").Elements().Child("invite").Attributes().Get("to")
	invJID, _ := jid.NewWithString(invJIDStr, true)
	return invJID
}

func (s *Muc) sendPM(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	// private message should be addressed to a particular occupant, not the whole room
	if !message.ToJID().IsFull() {
		_ = s.router.Route(ctx, message.BadRequestError())
		return
	}

	// check if user is allowed to send the pm
	if !s.userCanPMOccupant(ctx, room, message.FromJID(), message.ToJID(), message) {
		return
	}

	// send the PM
	sendersOccupantJID, _ := room.UserToOccupant[*message.FromJID().ToBareJID()]
	s.messageOccupant(ctx, message.ToJID(), &sendersOccupantJID, message)
}

func (s *Muc) userCanPMOccupant(ctx context.Context, room *mucmodel.Room, usrJID, occJID *jid.JID, message *xmpp.Message) bool {
	// check if user is in the room
	usrOccJID, found := room.UserToOccupant[*usrJID.ToBareJID()]
	if !found {
		_ = s.router.Route(ctx, message.NotAcceptableError())
		return false
	}

	// check if user can send private messages in this room
	usrOcc, err := s.repOccupant.FetchOccupant(ctx, &usrOccJID)
	if err != nil || usrOcc == nil {
		_ = s.router.Route(ctx, message.InternalServerError())
		return false
	}
	if !room.Config.OccupantCanSendPM(usrOcc) {
		_ = s.router.Route(ctx, message.NotAcceptableError())
		return false
	}

	// check if the target occupant exists
	occ, err := s.repOccupant.FetchOccupant(ctx, occJID)
	if err != nil || occ == nil {
		_ = s.router.Route(ctx, message.ItemNotFoundError())
		return false
	}

	return true
}

func (s *Muc) messageEveryone(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	// the groupmessage should be addressed to the whole room, not a particular occupant
	if message.ToJID().IsFull() {
		_ = s.router.Route(ctx, message.BadRequestError())
		return
	}

	// check if user is allowed to send a groupchat message
	if !s.userHasVoice(ctx, room, message.FromJID(), message) {
		return
	}

	sendersOccupantJID, _ := room.UserToOccupant[*message.FromJID().ToBareJID()]
	for _, occJID := range room.UserToOccupant {
		s.messageOccupant(ctx, &occJID, &sendersOccupantJID, message)
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
