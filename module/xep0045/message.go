/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func isVoiceRequest(message *xmpp.Message) bool {
	x := message.Elements().Child("x")
	if x == nil || x.Namespace() != "jabber:x:data" || x.Type() != "submit" {
		return false
	}
	return true
}

func (s *Muc) voiceRequest(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	occ, errStanza := s.getOccupantFromStanza(ctx, room, message)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	if !room.Config.Moderated {
		_ = s.router.Route(ctx, message.NotAllowedError())
		return
	}

	switch {
	case occ.IsVisitor():
		errStanza = s.askForVoice(ctx, room, occ, message)
		if errStanza != nil {
			_ = s.router.Route(ctx, errStanza)
		}
	case occ.IsModerator():
		errStanza = s.approveVoiceRequest(ctx, room, message)
		if errStanza != nil {
			_ = s.router.Route(ctx, errStanza)
		}
	}
}

func (s *Muc) askForVoice(ctx context.Context, room *mucmodel.Room, visitor *mucmodel.Occupant,
	message *xmpp.Message) xmpp.Stanza {
	formEl := message.Elements().Child("x")
	form, err := xep0004.NewFormFromElement(formEl)
	if err != nil {
		return message.BadRequestError()
	}

	if form.Fields.ValueForFieldOfType("muc#role", xep0004.ListSingle) != "participant" {
		return message.NotAllowedError()
	}

	approvalForm := s.getVoiceRequestForm(ctx, visitor)
	msg := xmpp.NewElementName("message").SetID(uuid.New().String())
	msg.AppendElement(approvalForm.Element())
	for _, occJID := range room.GetAllOccupantJIDs() {
		o, _ := s.repOccupant.FetchOccupant(ctx, &occJID)
		if o.IsModerator() {
			s.sendMessageToOccupant(ctx, o, room.RoomJID, msg)
		}
	}

	return nil
}

func (s *Muc) approveVoiceRequest(ctx context.Context, room *mucmodel.Room,
	message *xmpp.Message) xmpp.Stanza {
	formEl := message.Elements().Child("x")
	form, err := xep0004.NewFormFromElement(formEl)
	if err != nil {
		return message.BadRequestError()
	}

	occ, err := s.processVoiceApprovalForm(ctx, room, form)
	if err != nil {
		return message.BadRequestError()
	}
	if occ != nil {
		presenceEl := getOccupantChangeElement(occ, "")
		s.sendPresenceToRoom(ctx, room, occ.OccupantJID, presenceEl)
	}

	return nil
}

func (s *Muc) processVoiceApprovalForm(ctx context.Context, room *mucmodel.Room,
	form *xep0004.DataForm) (*mucmodel.Occupant, error) {
	requestAllow := false
	var role, userJIDStr, nick string
	for _, field := range form.Fields {
		if len(field.Values) == 0 {
			continue
		}
		switch field.Var {
		case "muc#role":
			role = field.Values[0]
		case "muc#jid":
			userJIDStr = field.Values[0]
		case "muc#roomnick":
			nick = field.Values[0]
		case "muc#request_allow":
			requestAllow, _ = strconv.ParseBool(field.Values[0])
		}
	}

	if requestAllow {
		occ, err := s.approveVoice(ctx, room, userJIDStr, role, nick)
		if err != nil {
			return nil, err
		}
		return occ, nil
	}

	return nil, nil
}

func (s *Muc) approveVoice(ctx context.Context, room *mucmodel.Room, userJIDStr, role,
	nick string) (*mucmodel.Occupant, error) {
	userJID, err := jid.NewWithString(userJIDStr, false)
	if err != nil {
		return nil, err
	}
	occJID, ok := room.GetOccupantJID(userJID.ToBareJID())
	if !ok {
		return nil, fmt.Errorf("User not in the room")
	}
	o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		return nil, err
	}
	if role != "participant" || nick != o.OccupantJID.Resource() {
		return nil, fmt.Errorf("Form not filled out correctly")
	}

	o.SetRole("participant")
	err = s.repOccupant.UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (s *Muc) changeSubject(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	occ, errStanza := s.getOccupantFromStanza(ctx, room, message)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	newSubject := message.Elements().Child("subject").Text()
	if room.Config.OccupantCanChangeSubject(occ) {
		room.Subject = newSubject
		s.repRoom.UpsertRoom(ctx, room)
	} else {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return
	}
	subjectEl := xmpp.NewElementName("subject").SetText(newSubject)
	msgEl := xmpp.NewElementName("message").SetType("groupchat").SetID(uuid.New().String())
	msgEl.AppendElement(subjectEl)
	s.sendMessageToRoom(ctx, room, occ.OccupantJID, msgEl)
}

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
	if !room.UserIsInvited(message.FromJID().ToBareJID()) {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return
	}

	room.DeleteInvite(message.FromJID().ToBareJID())
	s.repRoom.UpsertRoom(ctx, room)

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
	if errStanza := s.userHasVoice(ctx, room, message.FromJID(), message); errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	occ, errStanza := s.getOccupantFromStanza(ctx, room, message)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	if !room.Config.AllowInvites || (!room.Config.Open && !occ.IsModerator()) {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return
	}

	// add to the list of invited users
	invJID := getInvitedUserJID(message)
	err := room.InviteUser(invJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, message.InternalServerError())
	}
	s.repRoom.UpsertRoom(ctx, room)

	s.forwardInviteToUser(ctx, room, message)
}

func (s *Muc) forwardInviteToUser(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	inviteFrom := message.FromJID()
	inviteTo := getInvitedUserJID(message)

	msg := getInvitationStanza(room, inviteFrom, inviteTo, message)
	_ = s.router.Route(ctx, msg)
}

func (s *Muc) sendPM(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	// private message should be addressed to a particular occupant, not the whole room
	if !message.ToJID().IsFull() {
		_ = s.router.Route(ctx, message.BadRequestError())
		return
	}

	// check if user is allowed to send the pm
	if errStanza := s.userCanPMOccupant(ctx, room, message.FromJID(), message.ToJID(), message); errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	// send the PM
	senderJID, ok := room.GetOccupantJID(message.FromJID().ToBareJID())
	if !ok {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return
	}

	msgBody := message.Elements().Child("body")
	if msgBody == nil {
		_ = s.router.Route(ctx, message.BadRequestError())
		return
	}

	s.messageOccupant(ctx, message.ToJID(), &senderJID, msgBody, message.ID(), true)
}

func (s *Muc) userCanPMOccupant(ctx context.Context, room *mucmodel.Room, usrJID, occJID *jid.JID, message *xmpp.Message) xmpp.Stanza {
	// check if user can send private messages in this room
	usrOccJID, ok := room.GetOccupantJID(usrJID.ToBareJID())
	if !ok {
		return message.NotAcceptableError()
	}

	usrOcc, err := s.repOccupant.FetchOccupant(ctx, &usrOccJID)
	if err != nil || usrOcc == nil {
		return message.InternalServerError()
	}

	if !room.Config.OccupantCanSendPM(usrOcc) {
		return message.NotAcceptableError()
	}

	// check if the target occupant exists
	occ, err := s.repOccupant.FetchOccupant(ctx, occJID)
	if err != nil || occ == nil {
		return message.ItemNotFoundError()
	}

	// make sure the target occupant is in the same room
	if occJID.ToBareJID().String() != room.RoomJID.String() {
		return message.NotAcceptableError()
	}

	return nil
}

func (s *Muc) messageEveryone(ctx context.Context, room *mucmodel.Room, message *xmpp.Message) {
	// the groupmessage should be addressed to the whole room, not a particular occupant
	if message.ToJID().IsFull() {
		_ = s.router.Route(ctx, message.BadRequestError())
		return
	}

	// check if user is allowed to send a groupchat message
	if errStanza := s.userHasVoice(ctx, room, message.FromJID(), message); errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	sendersOccupantJID, ok := room.GetOccupantJID(message.FromJID().ToBareJID())
	if !ok {
		_ = s.router.Route(ctx, message.ForbiddenError())
		return
	}

	msgBody := message.Elements().Child("body")
	if msgBody == nil {
		_ = s.router.Route(ctx, message.BadRequestError())
		return
	}

	for _, occJID := range room.GetAllOccupantJIDs() {
		s.messageOccupant(ctx, &occJID, &sendersOccupantJID, msgBody, message.ID(), false)
	}
}

func (s *Muc) userHasVoice(ctx context.Context, room *mucmodel.Room, userJID *jid.JID,
	message *xmpp.Message) xmpp.Stanza {
	// user has to be occupant of the room
	occJID, ok := room.GetOccupantJID(userJID.ToBareJID())
	if !ok {
		return message.NotAcceptableError()
	}

	occ, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		return message.InternalServerError()
	}

	if room.Config.Moderated && occ.IsVisitor() {
		return message.ForbiddenError()
	}

	return nil
}

func (s *Muc) messageOccupant(ctx context.Context, occJID, senderJID *jid.JID,
	body xmpp.XElement, id string, private bool) {
	occupant, err := s.repOccupant.FetchOccupant(ctx, occJID)
	if err != nil {
		log.Error(err)
		return
	}

	msgEl := getMessageElement(body, id, private)
	_ = s.sendMessageToOccupant(ctx, occupant, senderJID, msgEl)
}
