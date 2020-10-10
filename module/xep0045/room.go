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
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	initialRoomConfigInstructions = `
Your room has been created!
To accept the default configuration, click OK. To
select a different configuration, please complete
this form.
`

	roomConfigInstructions = "Complete this form to modify the configuration of your room"

	ConfigName = "muc#roomconfig_roomname"

	ConfigDesc = "muc#roomconfig_roomdesc"

	ConfigAllowPM = "muc#roomconfig_allowpm"

	ConfigAllowInvites = "muc#roomconfig_allowinvites"

	ConfigChangeSubj = "muc#roomconfig_changesubject"

	ConfigMemberList = "muc#roomconfig_getmemberlist"

	ConfigLanguage = "muc#roomconfig_lang"

	ConfigMaxUsers = "muc#roomconfig_maxusers"

	ConfigMembersOnly = "muc#roomconfig_membersonly"

	ConfigModerated = "muc#roomconfig_moderatedroom"

	ConfigPwdProtected = "muc#roomconfig_passwordprotectedroom"

	ConfigPersistent = "muc#roomconfig_persistentroom"

	ConfigPublic = "muc#roomconfig_publicroom"

	ConfigAdmins = "muc#roomconfig_roomadmins"

	ConfigOwners = "muc#roomconfig_roomowners"

	ConfigPwd = "muc#roomconfig_roomsecret"

	ConfigPubSub = "muc#roomconfig_pubsub"

	ConfigWhoIs = "muc#roomconfig_whois"
)

func (s *Muc) newRoom(ctx context.Context, ownerFullJID, ownerOccJID *jid.JID) error {
	roomJID := ownerOccJID.ToBareJID()

	owner, err := s.createOwner(ctx, ownerFullJID, ownerOccJID)
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
		InvitedUsers:   make(map[jid.JID]bool),
		Locked:         true,
	}

	r.AddOccupant(owner)
	err := s.repRoom.UpsertRoom(ctx, r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func userIsRoomMember(room *mucmodel.Room, occupant *mucmodel.Occupant, userJID *jid.JID) bool {
	_, invited := room.InvitedUsers[*userJID]
	if invited {
		return true
	}

	if occupant.IsOwner() || occupant.IsAdmin() || occupant.IsMember() {
		return true
	}

	return false
}

func (s *Muc) GetRoomAdmins(ctx context.Context, r *mucmodel.Room) []string {
	admins := make([]string, 0)
	for _, occJID := range r.UserToOccupant {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			log.Error(err)
			return nil
		}
		if o.IsAdmin() {
			admins = append(admins, occJID.String())
		}
	}
	return admins
}

func (s *Muc) GetRoomOwners(ctx context.Context, r *mucmodel.Room) []string {
	owners := make([]string, 0)
	for bareJID, occJID := range r.UserToOccupant {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			log.Error(err)
			return nil
		}
		if o.IsOwner() {
			owners = append(owners, bareJID.String())
		}
	}
	return owners
}

func (s *Muc) SetRoomAdmin(ctx context.Context, room *mucmodel.Room, adminJID *jid.JID) error {
	// check if the occupant is in the room
	occJID, found := room.UserToOccupant[*adminJID]
	if !found {
		return fmt.Errorf("muc: user has to enter the room before it can be made admin")
	}

	occupant, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		return err
	}

	err = occupant.SetAffiliation("admin")
	if err != nil {
		return err
	}

	err = s.repOccupant.UpsertOccupant(ctx, occupant)
	if err != nil {
		return err
	}

	return nil
}

func (s *Muc) SetRoomOwner(ctx context.Context, room *mucmodel.Room, ownerJID *jid.JID) error {
	// check if the occupant is in the room
	occJID, found := room.UserToOccupant[*ownerJID]
	if !found {
		return fmt.Errorf("muc: user has to enter the room before it can be made owner")
	}

	occupant, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		return err
	}

	err = occupant.SetAffiliation("owner")
	if err != nil {
		return err
	}

	err = s.repOccupant.UpsertOccupant(ctx, occupant)
	if err != nil {
		return err
	}

	return nil
}

func (s *Muc) AddOccupantToRoom(ctx context.Context, room *mucmodel.Room, occupant *mucmodel.Occupant) error {
	room.AddOccupant(occupant)

	err := s.repOccupant.UpsertOccupant(ctx, occupant)
	if err != nil {
		return err
	}

	return s.repRoom.UpsertRoom(ctx, room)
}

func (s *Muc) getRoomConfigForm(ctx context.Context, room *mucmodel.Room) *xep0004.DataForm {
	form := &xep0004.DataForm{
		Type:         xep0004.Form,
		Title:        "Configuration for " + room.Name + "Room",
		Instructions: getRoomConfigInstructions(room),
	}
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    xep0004.FormType,
		Type:   xep0004.Hidden,
		Values: []string{"http://jabber.org/protocol/muc#roomconfig"},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigName,
		Type:   xep0004.TextSingle,
		Label:  "Natural-Language Room Name",
		Values: []string{room.Name},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigDesc,
		Type:   xep0004.TextSingle,
		Label:  "Short description of Room",
		Values: []string{room.Desc},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigLanguage,
		Type:   xep0004.TextSingle,
		Label:  "Natural Language for Room Discussion",
		Values: []string{room.Language},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigChangeSubj,
		Type:   xep0004.Boolean,
		Label:  "Allow Occupants to Change Subject?",
		Values: []string{boolToStr(room.Config.AllowSubjChange)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigAllowInvites,
		Type:   xep0004.Boolean,
		Label:  "Allow Occupants to Invite Others?",
		Values: []string{boolToStr(room.Config.AllowInvites)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigMembersOnly,
		Type:   xep0004.Boolean,
		Label:  "Make Room Members Only?",
		Values: []string{boolToStr(!room.Config.Open)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigModerated,
		Type:   xep0004.Boolean,
		Label:  "Make Room Moderated?",
		Values: []string{boolToStr(room.Config.Moderated)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigPersistent,
		Type:   xep0004.Boolean,
		Label:  "Make Room Persistent?",
		Values: []string{boolToStr(room.Config.Persistent)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigPublic,
		Type:   xep0004.Boolean,
		Label:  "Make Room Publicly Searchable?",
		Values: []string{boolToStr(room.Config.Public)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigWhoIs,
		Type:   xep0004.Boolean,
		Label:  "Make room NonAnonymous? (show real JIDs)",
		Values: []string{boolToStr(room.Config.NonAnonymous)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigPwdProtected,
		Type:   xep0004.Boolean,
		Label:  "Password Required to Enter?",
		Values: []string{boolToStr(room.Config.PwdProtected)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type:   xep0004.Fixed,
		Values: []string{"If the password is required to enter the room, specify it below"},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigPwd,
		Type:   xep0004.TextSingle,
		Label:  "Password",
		Values: []string{room.Config.Password},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigAllowPM,
		Type:   xep0004.ListSingle,
		Label:  "Roles that May Send Private Messages",
		Values: []string{room.Config.GetSendPM()},
		Options: []xep0004.Option{
			xep0004.Option{Label: "Anyone", Value: mucmodel.All},
			xep0004.Option{Label: "Moderators Only", Value: mucmodel.Moderators},
			xep0004.Option{Label: "Nobody", Value: mucmodel.None},
		},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigMemberList,
		Type:   xep0004.ListSingle,
		Label:  "Who Can Retrieve Member List",
		Values: []string{room.Config.GetCanGetMemberList()},
		Options: []xep0004.Option{
			xep0004.Option{Label: "Anyone", Value: mucmodel.All},
			xep0004.Option{Label: "Moderators Only", Value: mucmodel.Moderators},
			xep0004.Option{Label: "Nobody", Value: mucmodel.None},
		},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigMaxUsers,
		Type:   xep0004.ListSingle,
		Label:  "Maximum Number of Occupants (-1 for unlimited)",
		Values: []string{strconv.Itoa(room.Config.MaxOccCnt)},
		Options: []xep0004.Option{
			xep0004.Option{Label: "10", Value: "10"},
			xep0004.Option{Label: "20", Value: "20"},
			xep0004.Option{Label: "30", Value: "30"},
			xep0004.Option{Label: "50", Value: "50"},
			xep0004.Option{Label: "100", Value: "100"},
			xep0004.Option{Label: "500", Value: "100"},
			xep0004.Option{Label: "-1", Value: "-1"},
		},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigAdmins,
		Type:   xep0004.JidMulti,
		Label:  "Full List of Room Admins",
		Values: s.GetRoomAdmins(ctx, room),
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigOwners,
		Type:   xep0004.JidMulti,
		Label:  "Full List of Room Owners",
		Values: s.GetRoomOwners(ctx, room),
	})
	return form
}

func getRoomConfigInstructions(room *mucmodel.Room) (instr string) {
	if room.Locked {
		instr = initialRoomConfigInstructions
	} else {
		instr = roomConfigInstructions
	}
	return
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
				log.Error(err)
				ok = false
			}
			room.Config.AllowSubjChange = n
		case ConfigAllowInvites:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.AllowInvites = n
		case ConfigMembersOnly:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.Open = n
		case ConfigModerated:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.Moderated = n
		case ConfigPersistent:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.Persistent = n
		case ConfigPublic:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.Public = n
		case ConfigPwdProtected:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.PwdProtected = n
		case ConfigPwd:
			room.Config.Password = field.Values[0]
		case ConfigAllowPM:
			err := room.Config.SetWhoCanSendPM(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
		case ConfigMemberList:
			err := room.Config.SetWhoCanGetMemberList(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
		case ConfigWhoIs:
			n, err := strconv.ParseBool(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.NonAnonymous = n
		case ConfigMaxUsers:
			n, err := strconv.Atoi(field.Values[0])
			if err != nil {
				log.Error(err)
				ok = false
			}
			room.Config.MaxOccCnt = n
		case ConfigAdmins:
			for _, j := range field.Values {
				if j == "" {
					continue
				}
				bareJID, err := jid.NewWithString(j, false)
				if err != nil {
					log.Error(err)
					ok = false
				}
				err = s.SetRoomAdmin(ctx, room, bareJID)
				if err != nil {
					log.Error(err)
					ok = false
				}
			}
		case ConfigOwners:
			for _, j := range field.Values {
				if j == "" {
					continue
				}
				bareJID, err := jid.NewWithString(j, false)
				if err != nil {
					log.Error(err)
					ok = false
				}
				err = s.SetRoomOwner(ctx, room, bareJID)
				if err != nil {
					log.Error(err)
					ok = false
				}
			}
		}
	}

	// the password has to be specified if it is required to enter the room
	if room.Config.PwdProtected && room.Config.Password == "" {
		log.Infof("Password required but not supplied by the room owner")
		ok = false
	}

	if ok {
		room.Locked = false
		s.repRoom.UpsertRoom(ctx, room)
	}

	return ok
}

func boolToStr(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
