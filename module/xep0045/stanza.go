/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
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
)

const (
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

func getPasswordFromPresence(presence *xmpp.Presence) string {
	x := presence.Elements().ChildNamespace("x", mucNamespace)
	if x == nil {
		return ""
	}
	pwd := x.Elements().Child("password")
	if pwd == nil {
		return ""
	}
	return pwd.Text()
}

func getOccupantStatusStanza(o *mucmodel.Occupant, to *jid.JID, includeUserJID bool) xmpp.Stanza {
	x := newOccupantAffiliationRoleElement(o, includeUserJID)
	el := xmpp.NewElementName("presence").AppendElement(x).SetID(uuid.New().String())

	p, err := xmpp.NewPresenceFromElement(el, o.OccupantJID, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	return p
}

func getOccupantSelfPresenceStanza(o *mucmodel.Occupant, to *jid.JID, nonAnonymous bool, id string) xmpp.Stanza {
	x := newOccupantAffiliationRoleElement(o, false).AppendElement(newStatusElement("110"))
	if nonAnonymous {
		x.AppendElement(newStatusElement("100"))
	}
	el := xmpp.NewElementName("presence").AppendElement(x).SetID(id)
	p, err := xmpp.NewPresenceFromElement(el, o.OccupantJID, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	return p
}

func getRoomSubjectStanza(subject string, from, to *jid.JID) xmpp.Stanza {
	s := xmpp.NewElementName("subject").SetText(subject)
	m := xmpp.NewElementName("message").SetType("groupchat").SetID(uuid.New().String())
	m.AppendElement(s)
	message, err := xmpp.NewMessageFromElement(m, from, to)
	if err != nil{
		log.Error(err)
		return nil
	}
	return message
}

func getAckStanza(from, to *jid.JID) xmpp.Stanza {
	e := xmpp.NewElementNamespace("x", mucNamespaceUser)
	e.AppendElement(newItemElement("owner", "moderator"))
	e.AppendElement(newStatusElement("110"))
	e.AppendElement(newStatusElement("210"))

	presence := xmpp.NewElementName("presence").AppendElement(e)
	ack, err := xmpp.NewPresenceFromElement(presence, from, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	return ack
}

func getFormStanza(iq *xmpp.IQ, form *xep0004.DataForm) xmpp.Stanza {
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner)
	query.AppendElement(form.Element())

	e := xmpp.NewElementName("iq").SetID(iq.ID()).SetType("result").AppendElement(query)
	stanza, err := xmpp.NewIQFromElement(e, iq.ToJID(), iq.FromJID())
	if err != nil {
		log.Error(err)
		return nil
	}
	return stanza
}

func newItemElement(affiliation, role string) *xmpp.Element {
	i := xmpp.NewElementName("item")
	if affiliation == "" {
		affiliation = "none"
	}
	if role == "" {
		role = "none"
	}
	i.SetAttribute("affiliation", affiliation)
	i.SetAttribute("role", role)
	return i
}

func newStatusElement(code string) *xmpp.Element {
	s := xmpp.NewElementName("status")
	s.SetAttribute("code", code)
	return s
}

func newOccupantAffiliationRoleElement(o *mucmodel.Occupant, includeUserJID bool) *xmpp.Element {
	item := newItemElement(o.GetAffiliation(), o.GetRole())
	if includeUserJID {
		item.SetAttribute("jid", o.BareJID.String())
	}
	e := xmpp.NewElementNamespace("x", mucNamespaceUser)
	e.AppendElement(item)
	return e
}

func isIQForInstantRoomCreate(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	if query == nil {
		return false
	}
	if query.Namespace() != mucNamespaceOwner || query.Elements().Count() != 1 {
		return false
	}
	x := query.Elements().Child("x")
	if x == nil {
		return false
	}
	if x.Namespace() != "jabber:x:data" || x.Type() != "submit" || x.Elements().Count() != 0 {
		return false
	}
	return true
}

func isIQForRoomConfigRequest(iq *xmpp.IQ) bool {
	if !iq.IsGet() {
		return false
	}
	query := iq.Elements().Child("query")
	if query == nil {
		return false
	}
	if query.Namespace() != mucNamespaceOwner || query.Elements().Count() != 0 {
		return false
	}
	return true
}

func isIQForRoomConfigSubmission(iq *xmpp.IQ) bool {
	if !iq.IsSet() {
		return false
	}
	query := iq.Elements().Child("query")
	if query == nil {
		return false
	}
	if query.Namespace() != mucNamespaceOwner || query.Elements().Count() != 1 {
		return false
	}
	form := query.Elements().Child("x")
	if form == nil || form.Namespace() != xep0004.FormNamespace || form.Type() != "submit" {
		return false
	}
	return true
}

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

func boolToStr(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
