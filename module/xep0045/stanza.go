/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
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
