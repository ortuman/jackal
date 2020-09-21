/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"strconv"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

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

	occupant, err := s.repo.Occupant().FetchOccupant(ctx, occupantJID)
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
		isMember := userIsMember(room, occupant, userJID.ToBareJID())
		if !isMember {
			return presence.RegistrationRequiredError()
		}
	}
	return nil
}

func (s *Muc) sendEnterRoomAck(ctx context.Context, room *mucmodel.Room, presence *xmpp.Presence) error {
	newOccupant, err := s.repo.Occupant().FetchOccupant(ctx, presence.ToJID())
	if err != nil {
		return err
	}

	for usrJID, occJID := range room.UserToOccupant {
		// skip the user entering the room
		if usrJID.String() == newOccupant.BareJID.String() {
			continue
		}
		o, err := s.repo.Occupant().FetchOccupant(ctx, &occJID)
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

func (s *Muc) newRoom(ctx context.Context, userJID, occJID *jid.JID) error {
	roomJID := occJID.ToBareJID()

	owner, err := s.createOwner(ctx, userJID, occJID)
	if err != nil {
		return err
	}

	_, err = s.createRoom(ctx, roomJID, owner)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.allRooms = append(s.allRooms, *roomJID)
	s.mu.Unlock()

	return nil
}

func (s *Muc) createRoom(ctx context.Context, roomJID *jid.JID, owner *mucmodel.Occupant) (*mucmodel.Room, error) {
	r := &mucmodel.Room{
		Config:         s.GetDefaultRoomConfig(),
		Name:           roomJID.Node(),
		RoomJID:        roomJID,
		UserToOccupant: make(map[jid.JID]jid.JID),
		Locked:         true,
	}

	r.AddOccupant(owner)
	err := s.repo.Room().UpsertRoom(ctx, r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Muc) sendRoomCreateAck(ctx context.Context, from, to *jid.JID) error {
	el := getAckStanza(from, to)
	err := s.router.Route(ctx, el)
	return err
}

func (s *Muc) createInstantRoom(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	_, ok := s.getOwnerFromIQ(ctx, room, iq)
	if !ok {
		return
	}
	room.Locked = false
	err := s.repo.Room().UpsertRoom(ctx, room)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
	}
	_ = s.router.Route(ctx, iq.ResultIQ())
}

func (s *Muc) sendRoomConfiguration(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	_, ok := s.getOwnerFromIQ(ctx, room, iq)
	if !ok {
		return
	}
	configForm := s.getRoomConfigForm(ctx, room)
	stanza := getFormStanza(iq, configForm)
	err := s.router.Route(ctx, stanza)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.BadRequestError())
	}
}

func (s *Muc) processRoomConfiguration(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	_, ok := s.getOwnerFromIQ(ctx, room, iq)
	if !ok {
		return
	}

	form, err := xep0004.NewFormFromElement(iq.Elements().Child("query").Elements().Child("x"))
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.BadRequestError())
		return
	}

	ok = s.updateRoomWithForm(ctx, room, form)
	if !ok {
		_ = s.router.Route(ctx, iq.NotAcceptableError())
		return
	}

	_ = s.router.Route(ctx, iq.ResultIQ())
}

func (s *Muc) updateRoomWithForm(ctx context.Context, room *mucmodel.Room, form *xep0004.DataForm) (ok bool) {
	ok = true
	for _, field := range form.Fields {
		if len(field.Values) == 0 {
			continue
		}
		switch field.Var {
		case ConfigName:
			room.Name = field.Values[0]
		case ConfigDesc:
			room.Desc = field.Values[0]
		case ConfigLanguage:
			room.Language = field.Values[0]
		case ConfigChangeSubj:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.AllowSubjChange = n
		case ConfigAllowInvites:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.AllowInvites = n
		case ConfigMembersOnly:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.Open = n
		case ConfigModerated:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.Moderated = n
		case ConfigPersistent:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.Persistent = n
		case ConfigPublic:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.Public = n
		case ConfigPwdProtected:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.PwdProtected = n
		case ConfigPwd:
			room.Config.Password = field.Values[0]
		case ConfigAllowPM:
			err := room.Config.SetWhoCanSendPM(field.Values[0])
			if err != nil {
				ok = false
			}
		case ConfigMemberList:
			err := room.Config.SetWhoCanGetMemberList(field.Values[0])
			if err != nil {
				ok = false
			}
		case ConfigWhoIs:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.NonAnonymous = n
		case ConfigMaxUsers:
			n, err := strconv.Atoi(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.MaxOccCnt = n
		case ConfigAdmins:
			for _, j := range field.Values {
				bareJID, err := jid.NewWithString(j, false)
				if err != nil {
					ok = false
				}
				err = s.SetRoomAdmin(ctx, room, bareJID)
				if err != nil {
					ok = false
				}
			}
		case ConfigOwners:
			for _, j := range field.Values {
				bareJID, err := jid.NewWithString(j, false)
				if err != nil {
					ok = false
				}
				err = s.SetRoomOwner(ctx, room, bareJID)
				if err != nil {
					ok = false
				}
			}
		}
	}

	// the password has to be specified if it is required to enter the room
	if room.Config.PwdProtected && room.Config.Password == "" {
		ok = false
	}

	if ok {
		room.Locked = false
		s.repo.Room().UpsertRoom(ctx, room)
	}

	return ok
}

func (s *Muc) getOwnerFromIQ(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) (*mucmodel.Occupant, bool) {
	fromJID, err := jid.NewWithString(iq.From(), true)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.BadRequestError())
		return nil, false
	}
	occJID, ok := room.UserToOccupant[*fromJID.ToBareJID()]
	if !ok {
		_ = s.router.Route(ctx, iq.BadRequestError())
		return nil, false
	}
	occ, err := s.GetOccupant(ctx, &occJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.BadRequestError())
		return nil, false
	}
	if !occ.IsOwner() {
		_ = s.router.Route(ctx, iq.ForbiddenError())
		return nil, false
	}
	return occ, true
}

func userIsMember(room *mucmodel.Room, occupant *mucmodel.Occupant, userJID *jid.JID) bool {
	_, invited := room.InvitedUsers[*userJID]
	if invited {
		return true
	}

	if occupant.IsOwner() || occupant.IsAdmin() || occupant.IsMember() {
		return true
	}

	return false
}
