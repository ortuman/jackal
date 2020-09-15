/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func (s *Muc) newRoom(ctx context.Context, from, to *jid.JID, roomName, ownerNick string, locked bool) error {
	roomJID := to.ToBareJID()
	roomExists, _ := s.reps.Room().RoomExists(ctx, roomJID)
	if roomExists {
		return fmt.Errorf("Room %s already exists", roomName)
	}

	owner, err := s.createOwner(ctx, to, ownerNick, from)
	if err != nil {
		return err
	}
	_, err = s.createRoom(ctx, roomName, roomJID, owner, locked)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.allRooms = append(s.allRooms, *roomJID)
	s.mu.Unlock()
	return nil
}

func (s *Muc) createRoom(ctx context.Context, name string, roomJID *jid.JID, owner *mucmodel.Occupant, locked bool) (*mucmodel.Room, error) {
	r := &mucmodel.Room{
		Config:         s.GetDefaultRoomConfig(),
		Name:           name,
		RoomJID:        roomJID,
		UserToOccupant: make(map[jid.JID]jid.JID),
		Locked:         locked,
	}
	r.AddOccupant(owner)
	err := s.reps.Room().UpsertRoom(ctx, r)
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
	s.reps.Room().UpsertRoom(ctx, room)
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
		_ = s.router.Route(ctx, iq.BadRequestError())
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
		case ConfigHistory:
			n, err := strconv.Atoi(field.Values[0])
			if err != nil || n < 0 {
				ok = false
			}
			room.Config.HistCnt = n
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
		case ConfigEnableLogging:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				ok = false
			}
			room.Config.EnableLogging = n
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
			err := room.Config.SetWhoCanRealJIDDisc(field.Values[0])
			if err != nil {
				ok = false
			}
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
		s.reps.Room().UpsertRoom(ctx, room)
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
