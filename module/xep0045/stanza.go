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
