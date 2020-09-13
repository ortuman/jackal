/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"strconv"

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

	ConfigHistory = "muc#maxhistoryfetch"

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

	ConfigEnableLogging = "muc#roomconfig_enablelogging"

	ConfigPubSub = "muc#roomconfig_pubsub"

	ConfigPresenceBroadcast = "muc#roomconfig_presencebroadcast"

	ConfigWhoIs = "muc#roomconfig_whois"
)

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
	i.SetAttribute("affiliation", affiliation)
	i.SetAttribute("role", role)
	return i
}

func newStatusElement(code string) *xmpp.Element {
	s := xmpp.NewElementName("status")
	s.SetAttribute("code", code)
	return s
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

func getRoomConfigForm(room *mucmodel.Room) *xep0004.DataForm {
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
		Var:    ConfigHistory,
		Type:   xep0004.TextSingle,
		Label:  "Maximum Number of History Messages Returned by Room",
		Values: []string{strconv.Itoa(room.Config.HistCnt)},
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
		Var:    ConfigAllowInvites,
		Type:   xep0004.Boolean,
		Label:  "Allow Occupants to Invite Others?",
		Values: []string{boolToStr(room.Config.AllowInvites)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigEnableLogging,
		Type:   xep0004.Boolean,
		Label:  "Enable Public Logging?",
		Values: []string{boolToStr(room.Config.EnableLogging)},
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
		Var:   ConfigPubSub,
		Type:  xep0004.TextSingle,
		Label: "Associated pubsub node",
		// TODO this is the field that's not being used at the moment
		Values: []string{""},
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
			xep0004.Option{Label: "-1", Value: "-1"},
		},
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
		Var:    ConfigPwdProtected,
		Type:   xep0004.Boolean,
		Label:  "Password Required to Enter?",
		Values: []string{boolToStr(room.Config.PwdProtected)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigPersistent,
		Type:   xep0004.Boolean,
		Label:  "Make Room Persistent?",
		Values: []string{boolToStr(room.Config.Persistent)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		// TODO this field not used at the moment (it should not be boolean)
		Var:    ConfigPresenceBroadcast,
		Type:   xep0004.Boolean,
		Label:  "Roles for which Presence is Broadcasted?",
		Values: []string{"0"},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigPublic,
		Type:   xep0004.Boolean,
		Label:  "Make Room Publicly Searchable?",
		Values: []string{boolToStr(room.Config.Public)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type:   xep0004.Fixed,
		Values: []string{"If the password is required to enter the room, specify it below"},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigPwd,
		Type:   xep0004.TextSingle,
		Label:  "Password",
		Values: []string{boolToStr(room.Config.Public)},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigWhoIs,
		Type:   xep0004.ListSingle,
		Label:  "Who May Discover Real JIDs",
		Values: []string{room.Config.GetRealJIDDisc()},
		Options: []xep0004.Option{
			xep0004.Option{Label: "Anyone", Value: mucmodel.All},
			xep0004.Option{Label: "Moderators Only", Value: mucmodel.Moderators},
			xep0004.Option{Label: "Nobody", Value: mucmodel.None},
		},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigAdmins,
		Type:   xep0004.JidMulti,
		Label:  "Full List of Room Admins",
		Values: room.GetAdmins(),
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Var:    ConfigOwners,
		Type:   xep0004.JidMulti,
		Label:  "Full List of Room Owners",
		Values: room.GetOwners(),
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
