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

func isPresenceToEnterRoom(presence *xmpp.Presence) bool {
	if presence.Elements().Count() != 1 || presence.Type() != "" {
		return false
	}
	x := presence.Elements().ChildNamespace("x", mucNamespace)
	if x == nil || x.Text() != "" {
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
		log.Infof("muc: New room created, room JID is %s", presence.ToJID().ToBareJID().String())
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
	if occupant != nil && !occupant.IsOwner() && !occupant.IsAdmin() && room.Full() {
		_ = s.router.Route(ctx, presence.ServiceUnavailableError())
		return false, nil
	}

	return true, nil
}

func checkNicknameConflict(room *mucmodel.Room, newOccupant *mucmodel.Occupant,
	userJID, occupantJID *jid.JID, presence *xmpp.Presence) xmpp.Stanza {
	// check if the user, who is already in the room, is entering with a different nickname
	oJID, registered := room.UserToOccupant[*userJID.ToBareJID()]
	if registered && oJID.String() != occupantJID.String() {
		return presence.NotAcceptableError()
	}

	// check if another user is trying to use an already occupied nickname
	if !registered && newOccupant != nil {
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

	for usrJID, occJID := range room.UserToOccupant {
		// skip the user entering the room
		if usrJID.String() == newOccupant.BareJID.String() {
			continue
		}
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			return err
		}
		// notify the user of the new occupant
		p := getOccupantStatusStanza(o, presence.FromJID(),
			room.Config.OccupantCanDiscoverRealJID(o))
		_ = s.router.Route(ctx, p)

		// notify the new occupant about the user
		p = getOccupantStatusStanza(newOccupant, &usrJID,
			room.Config.OccupantCanDiscoverRealJID(newOccupant))
		_ = s.router.Route(ctx, p)
	}

	// final notification to the new occupant with status codes (self-presence)
	p := getOccupantSelfPresenceStanza(newOccupant, newOccupant.BareJID, room.Config.NonAnonymous,
		presence.ID())
	_ = s.router.Route(ctx, p)

	// send the room subject
	subj := getRoomSubjectStanza(room.Subject, room.RoomJID, presence.FromJID().ToBareJID())
	_ = s.router.Route(ctx, subj)

	return nil
}
