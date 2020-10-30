/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func (s *Muc) exitRoom(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) {
	o, errStanza := s.getOccupantFromPresence(ctx, room, presence)
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
		for _, occJID := range room.GetAllOccupantJIDs() {
			s.repOccupant.DeleteOccupant(ctx, &occJID)
		}
		s.repRoom.DeleteRoom(ctx, room.RoomJID)
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
		for _, resource := range o.GetAllResources() {
			to := addResourceToBareJID(o.BareJID, resource)
			p, err := xmpp.NewPresenceFromElement(resultPresence, occExiting.OccupantJID, to)
			if err != nil {
				return err
			}
			err = s.router.Route(ctx, p)
			if err != nil {
				return err
			}
		}
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
	o, errStanza := s.getOccupantFromPresence(ctx, room, presence)
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
		// TODO maybe rename this long name (occupantcandiscoverrealjid) to something smaller
		xEl := newOccupantAffiliationRoleElement(sender,
			room.Config.OccupantCanDiscoverRealJID(o), false)
		presence.AppendElement(xEl)
		for _, resource := range o.GetAllResources() {
			presence.SetID(uuid.New().String())
			to := addResourceToBareJID(o.BareJID, resource)
			p, err := xmpp.NewPresenceFromElement(presence, sender.OccupantJID, to)
			if err != nil {
				return err
			}
			_ = s.router.Route(ctx, p)
		}
	}

	return nil
}

func (s *Muc) changeNickname(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) {
	if errStanza := s.newNickIsAvailable(ctx, presence); errStanza != nil {
		_ = s.router.Route(ctx, errStanza)
		return
	}

	occ, errStanza := s.getOccupantFromPresence(ctx, room, presence)
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
		for _, resource := range o.GetAllResources() {
			to := addResourceToBareJID(o.BareJID, resource)

			// send unavailable stanza
			p := getOccupantUnavailableStanza(newOcc, oldJID, to, selfNotifying,
				room.Config.OccupantCanDiscoverRealJID(o))
			_ = s.router.Route(ctx, p)

			// send new status stanza
			p = getOccupantStatusStanza(newOcc, to, selfNotifying,
				room.Config.OccupantCanDiscoverRealJID(o))
			_ = s.router.Route(ctx, p)
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
	err = s.sendRoomCreateAck(ctx, presence.ToJID(), presence.FromJID())
	if err != nil {
		return err
	}
	return nil
}

func (s *Muc) sendRoomCreateAck(ctx context.Context, from, to *jid.JID) error {
	el := getAckStanza(from, to)
	err := s.router.Route(ctx, el)
	return err
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
		isMember := userIsRoomMember(room, occupant, userJID.ToBareJID())
		if !isMember {
			return presence.RegistrationRequiredError()
		}
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

	s.sendEnterRoomSuccess(ctx, room, newOccupant, presence.ID())

	return nil
}

func (s *Muc) sendPresenceAboutNewOccupant(ctx context.Context, room *mucmodel.Room,
	newOccupant, o *mucmodel.Occupant) {
	// notify the new occupant of the existing occupant
	for _, resource := range newOccupant.GetAllResources() {
		to := addResourceToBareJID(newOccupant.BareJID, resource)
		p := getOccupantStatusStanza(o, to, false,
			room.Config.OccupantCanDiscoverRealJID(newOccupant))
		_ = s.router.Route(ctx, p)
	}

	// notify the existing occupant of the new occupant
	for _, resource := range o.GetAllResources() {
		to := addResourceToBareJID(o.BareJID, resource)
		p := getOccupantStatusStanza(newOccupant, to, false,
			room.Config.OccupantCanDiscoverRealJID(o))
		_ = s.router.Route(ctx, p)
	}
}

func (s *Muc) sendEnterRoomSuccess(ctx context.Context, room *mucmodel.Room,
	newOccupant *mucmodel.Occupant, id string) {
	// final notification to the new occupant with status codes (self-presence)
	for _, resource := range newOccupant.GetAllResources() {
		to := addResourceToBareJID(newOccupant.BareJID, resource)
		p := getOccupantSelfPresenceStanza(newOccupant, to, room.Config.NonAnonymous, id)
		_ = s.router.Route(ctx, p)

		// send the room subject
		subj := getRoomSubjectStanza(room.Subject, room.RoomJID, to)
		_ = s.router.Route(ctx, subj)
	}
}
