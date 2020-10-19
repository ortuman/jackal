/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"github.com/google/uuid"
	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func getInvitedUserJID(message *xmpp.Message) *jid.JID {
	invJIDStr := message.Elements().Child("x").Elements().Child("invite").Attributes().Get("to")
	invJID, _ := jid.NewWithString(invJIDStr, true)
	return invJID
}


func getMessageElement(body xmpp.XElement, id string, private bool) *xmpp.Element {
	msgEl := xmpp.NewElementName("message").AppendElement(body)

	if id != "" {
		msgEl.SetID(id)
	} else {
		msgEl.SetID(uuid.New().String())
	}

	if private {
		msgEl.SetType("chat")
		msgEl.AppendElement(xmpp.NewElementNamespace("x", mucNamespaceUser))
	} else {
		msgEl.SetType("groupchat")
	}

	return msgEl
}

func getDeclineStanza(room *mucmodel.Room, message *xmpp.Message) xmpp.Stanza {
	toStr := message.Elements().Child("x").Elements().Child("decline").Attributes().Get("to")
	to, _ := jid.NewWithString(toStr, true)

	declineEl := xmpp.NewElementName("decline").SetAttribute("from",
		message.FromJID().ToBareJID().String())
	reasonEl := message.Elements().Child("x").Elements().Child("decline").Elements().Child("reason")
	if reasonEl != nil {
		declineEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(declineEl)
	msgEl := xmpp.NewElementName("message").AppendElement(xEl).SetID(message.ID())
	msg, err := xmpp.NewMessageFromElement(msgEl, room.RoomJID, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	return msg
}

func getInvitationStanza(room *mucmodel.Room, inviteFrom, inviteTo *jid.JID, message *xmpp.Message) xmpp.Stanza {
	inviteEl := xmpp.NewElementName("invite").SetAttribute("from", inviteFrom.String())
	reasonEl := message.Elements().Child("x").Elements().Child("invite").Elements().Child("reason")
	if reasonEl != nil {
		inviteEl.AppendElement(reasonEl)
	}
	xEl := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(inviteEl)
	if room.Config.PwdProtected {
		pwdEl := xmpp.NewElementName("password").SetText(room.Config.Password)
		xEl.AppendElement(pwdEl)
	}
	msgEl := xmpp.NewElementName("message").AppendElement(xEl).SetID(message.ID())
	msg, err := xmpp.NewMessageFromElement(msgEl, room.RoomJID, inviteTo)
	if err != nil {
		log.Error(err)
		return nil
	}
	return msg
}

func getOccupantUnavailableStanza(o *mucmodel.Occupant, from, to *jid.JID,
	selfNotifying, includeUserJID bool) xmpp.Stanza {
	// get the x element
	x := newOccupantAffiliationRoleElement(o, includeUserJID)

	// modifying the item element to include the nick
	itemEl := xmpp.NewElementFromElement(x.Elements().Child("item"))
	itemEl.SetAttribute("nick", o.OccupantJID.Resource())
	x.RemoveElements("item").AppendElement(itemEl)

	x.AppendElement(newStatusElement("303"))
	if selfNotifying {
		x.AppendElement(newStatusElement("110"))
	}

	el := xmpp.NewElementName("presence").AppendElement(x).SetID(uuid.New().String())
	el.SetType("unavailable")
	p, err := xmpp.NewPresenceFromElement(el, from, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	return p
}

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

func getOccupantStatusStanza(o *mucmodel.Occupant, to *jid.JID,
	selfNotifying, includeUserJID bool) xmpp.Stanza {
	x := newOccupantAffiliationRoleElement(o, includeUserJID)
	if selfNotifying {
		x.AppendElement(newStatusElement("110"))
	}
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
	if err != nil {
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

func addResourceToBareJID(bareJID *jid.JID, resource string) *jid.JID {
	res, _ := jid.NewWithString(bareJID.String()+"/"+resource, true)
	return res
}
