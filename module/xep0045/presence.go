/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"strings"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func (s *Muc) exitRoom(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) {
	o, errStanza := s.getOccupantFromStanza(ctx, room, presence)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	if o.OccupantJID.String() != presence.ToJID().String() {
		_ = s.router.Route(ctx, presence.ForbiddenError())
		return
	}

	s.occupantExitsRoom(ctx, room, o)

	err := s.sendOccExitedRoom(ctx, o, room)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, presence.InternalServerError())
	}
}

func (s *Muc) occupantExitsRoom(ctx context.Context, room *mucmodel.Room, o *mucmodel.Occupant) {
	if o.HasNoAffiliation() {
		s.repOccupant.DeleteOccupant(ctx, o.OccupantJID)
	} else {
		o.SetRole("")
		s.repOccupant.UpsertOccupant(ctx, o)
	}

	room.OccupantLeft(o)
	s.repRoom.UpsertRoom(ctx, room)

	if !room.Config.Persistent && room.IsEmpty() {
		s.deleteRoom(ctx, room)
	}
}

func (s *Muc) sendOccExitedRoom(ctx context.Context, occExiting *mucmodel.Occupant,
	room *mucmodel.Room) error {
	resultPresence := xmpp.NewElementName("presence").SetType("unavailable")

	for _, occJID := range room.GetAllOccupantJIDs() {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			return err
		}
		xEl := newOccupantAffiliationRoleElement(occExiting,
			room.Config.OccupantCanDiscoverRealJID(o), false)
		if occJID.String() == occExiting.OccupantJID.String() {
			xEl.AppendElement(newStatusElement("110"))
		}
		resultPresence.AppendElement(xEl)
		err = s.sendPresenceToOccupant(ctx, o, occExiting.OccupantJID, resultPresence)
	}
	return nil
}

func isChangingStatus(presence *xmpp.Presence) bool {
	status := presence.Elements().Child("status")
	show := presence.Elements().Child("show")
	if status == nil && show == nil {
		return false
	}
	return true
}

func (s *Muc) changeStatus(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) {
	o, errStanza := s.getOccupantFromStanza(ctx, room, presence)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	if o.OccupantJID.String() != presence.ToJID().String() {
		_ = s.router.Route(ctx, presence.ForbiddenError())
		return
	}

	if o.IsVisitor() {
		_ = s.router.Route(ctx, presence.ForbiddenError())
		return
	}

	show := presence.Elements().Child("show")
	status := presence.Elements().Child("status")
	err := s.sendStatus(ctx, room, o, show, status)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, presence.InternalServerError())
		return
	}
}

func (s *Muc) sendStatus(ctx context.Context, room *mucmodel.Room, sender *mucmodel.Occupant,
	show, status xmpp.XElement) error {
	presence := xmpp.NewElementName("presence").AppendElement(show).AppendElement(status)

	for _, occJID := range room.GetAllOccupantJIDs() {
		if occJID.String() == sender.OccupantJID.String() {
			continue
		}
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			return err
		}
		xEl := newOccupantAffiliationRoleElement(sender,
			room.Config.OccupantCanDiscoverRealJID(o), false)
		presence.AppendElement(xEl)
		err = s.sendPresenceToOccupant(ctx, o, sender.OccupantJID, presence)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Muc) changeNickname(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) {
	if errStanza := s.newNickIsAvailable(ctx, presence); errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	occ, errStanza := s.getOccupantFromStanza(ctx, room, presence)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}
	oldOccJID := occ.OccupantJID

	occ.SetAffiliation("")
	room.OccupantLeft(occ)
	s.repOccupant.DeleteOccupant(ctx, oldOccJID)

	occ.OccupantJID = presence.ToJID()
	room.AddOccupant(occ)
	s.repOccupant.UpsertOccupant(ctx, occ)
	s.repRoom.UpsertRoom(ctx, room)

	// send the unavailable and presence stanzas to the room members
	err := s.sendNickChangeAck(ctx, room, occ, oldOccJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, presence.InternalServerError())
		return
	}
}

func (s *Muc) sendNickChangeAck(ctx context.Context, room *mucmodel.Room,
	newOcc *mucmodel.Occupant, oldJID *jid.JID) error {
	for _, occJID := range room.GetAllOccupantJIDs() {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			return err
		}
		selfNotifying := (occJID.String() == newOcc.OccupantJID.String())
		getRealJID := room.Config.OccupantCanDiscoverRealJID(o)

		unavailableEl := getOccupantUnavailableElement(newOcc, selfNotifying, getRealJID)
		err = s.sendPresenceToOccupant(ctx, o, oldJID, unavailableEl)
		if err != nil {
			return err
		}

		statusEl := getOccupantStatusElement(newOcc, selfNotifying, getRealJID)
		err = s.sendPresenceToOccupant(ctx, o, newOcc.OccupantJID, statusEl)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Muc) newNickIsAvailable(ctx context.Context, presence *xmpp.Presence) xmpp.Stanza {
	o, err := s.repOccupant.FetchOccupant(ctx, presence.ToJID())
	if err != nil {
		log.Error(err)
		return presence.InternalServerError()
	}
	if o != nil {
		return presence.ConflictError()
	}
	return nil
}

func isPresenceToEnterRoom(presence *xmpp.Presence) bool {
	if presence.Type() != "" {
		return false
	}
	x := presence.Elements().ChildNamespace("x", mucNamespace)
	if x == nil || len(strings.TrimSpace(x.Text())) != 0 || x.Elements().Count() != 0 {
		return false
	}
	return true
}

func (s *Muc) enterRoom(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) {
	if room == nil {
		err := s.newRoomRequest(ctx, room, presence)
		if err != nil {
			_ = s.router.Route(ctx, presence.InternalServerError())
			return
		}
	} else {
		err := s.joinExistingRoom(ctx, room, presence)
		if err != nil {
			_ = s.router.Route(ctx, presence.InternalServerError())
			return
		}
	}
}

func (s *Muc) newRoomRequest(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) error {
	err := s.newRoom(ctx, presence.FromJID(), presence.ToJID())
	if err != nil {
		return err
	}

	el := getAckStanza(presence.ToJID(), presence.FromJID())
	_ = s.router.Route(ctx, el)
	return nil
}

func (s *Muc) joinExistingRoom(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) error {
	ok, err := s.occupantCanEnterRoom(ctx, room, presence)
	if !ok || err != nil {
		return err
	}

	occ, err := s.newOccupant(ctx, presence.FromJID(), presence.ToJID())
	if err != nil {
		return err
	}

	err = s.AddOccupantToRoom(ctx, room, occ)
	if err != nil {
		return err
	}

	err = s.sendEnterRoomAck(ctx, room, presence)
	if err != nil {
		return err
	}

	return nil
}

func (s *Muc) occupantCanEnterRoom(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) (bool, error) {
	userJID := presence.FromJID()
	occupantJID := presence.ToJID()

	occupant, err := s.repOccupant.FetchOccupant(ctx, occupantJID)
	if err != nil {
		return false, err
	}

	// no one can enter a locked room
	if room.Locked {
		_ = s.router.Route(ctx, presence.ItemNotFoundError())
		return false, nil
	}

	// nick for the occupant has to be provided
	if !occupantJID.IsFull() {
		_ = s.router.Route(ctx, presence.JidMalformedError())
		return false, nil
	}

	errStanza := checkNicknameConflict(room, occupant, userJID, occupantJID, presence)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return false, nil
	}

	errStanza = checkPassword(room, presence)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return false, nil
	}

	errStanza = checkOccupantMembership(room, occupant, userJID, presence)
	if errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return false, nil
	}

	// check if this occupant is banned
	if occupant != nil && occupant.IsOutcast() {
		_ = s.router.Route(ctx, presence.ForbiddenError())
		return false, nil
	}

	// check if the maximum number of occupants is reached
	if occupant != nil && !occupant.IsOwner() && !occupant.IsAdmin() && room.IsFull() {
		_ = s.router.Route(ctx, presence.ServiceUnavailableError())
		return false, nil
	}

	return true, nil
}

func checkNicknameConflict(room *mucmodel.Room, newOccupant *mucmodel.Occupant,
	userJID, occupantJID *jid.JID, presence *xmpp.Presence) xmpp.Stanza {
	// check if the user, who is already in the room, is entering with a different nickname
	oJID, ok := room.GetOccupantJID(userJID.ToBareJID())
	if ok && oJID.String() != occupantJID.String() {
		return presence.NotAcceptableError()
	}

	// check if another user is trying to use an already occupied nickname
	if newOccupant != nil && newOccupant.BareJID.String() != userJID.ToBareJID().String() {
		return presence.ConflictError()
	}

	return nil
}

func checkPassword(room *mucmodel.Room, presence *xmpp.Presence) xmpp.Stanza {
	// if password required, make sure that it is correctly supplied
	if room.Config.PwdProtected {
		pwd := getPasswordFromPresence(presence)
		if pwd != room.Config.Password {
			return presence.NotAuthorizedError()
		}
	}
	return nil
}

func checkOccupantMembership(room *mucmodel.Room, occupant *mucmodel.Occupant, userJID *jid.JID,
	presence *xmpp.Presence) xmpp.Stanza {
	// if members-only room, check that the occupant is a member
	if !room.Config.Open {
		if room.UserIsInvited(userJID.ToBareJID()) {
			return nil
		}
		if occupant != nil && !occupant.HasNoAffiliation() {
			return nil
		}
		return presence.RegistrationRequiredError()
	}
	return nil
}

func (s *Muc) sendEnterRoomAck(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) error {
	newOccupant, err := s.repOccupant.FetchOccupant(ctx, presence.ToJID())
	if err != nil {
		return err
	}

	for _, occJID := range room.GetAllOccupantJIDs() {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			return err
		}

		// skip the user entering the room
		if o.BareJID.String() == newOccupant.BareJID.String() {
			continue
		}

		s.sendPresenceAboutNewOccupant(ctx, room, newOccupant, o)
	}

	// final notification to the new occupant with status codes (self-presence)
	spEl := getOccupantSelfPresenceElement(newOccupant, room.Config.NonAnonymous, presence.ID())
	s.sendPresenceToOccupant(ctx, newOccupant, newOccupant.OccupantJID, spEl)

	// send the room subject
	subjEl := getRoomSubjectElement(room.Subject)
	s.sendMessageToOccupant(ctx, newOccupant, room.RoomJID, subjEl)

	return nil
}

func (s *Muc) sendPresenceAboutNewOccupant(ctx context.Context, room *mucmodel.Room,
	newOccupant, o *mucmodel.Occupant) {
	// notify the new occupant of the existing occupant
	oStatusEl := getOccupantStatusElement(o, false, room.Config.OccupantCanDiscoverRealJID(newOccupant))
	s.sendPresenceToOccupant(ctx, newOccupant, o.OccupantJID, oStatusEl)

	// notify the existing occupant of the new occupant
	newStatusEl := getOccupantStatusElement(newOccupant, false, room.Config.OccupantCanDiscoverRealJID(o))
	s.sendPresenceToOccupant(ctx, o, newOccupant.OccupantJID, newStatusEl)
}
