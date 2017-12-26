/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/xml"
)

const registerNamespace = "jabber:iq:register"

type XEPRegister struct {
	cfg        *config.ModRegistration
	strm       Stream
	registered bool
}

func NewXEPRegister(cfg *config.ModRegistration, strm Stream) *XEPRegister {
	return &XEPRegister{
		cfg:  cfg,
		strm: strm,
	}
}

func (x *XEPRegister) AssociatedNamespaces() []string {
	return []string{registerNamespace}
}

func (x *XEPRegister) MatchesIQ(iq *xml.IQ) bool {
	return iq.FindElementNamespace("query", registerNamespace) != nil
}

func (x *XEPRegister) ProcessIQ(iq *xml.IQ) {
	if !x.isValidToJid(iq.ToJID()) {
		x.strm.SendElement(iq.ForbiddenError())
		return
	}

	q := iq.FindElementNamespace("query", registerNamespace)
	if !x.strm.Authenticated() {
		if iq.IsGet() {
			// ...send registration fields to requester entity...
			x.sendRegistrationFields(iq, q)
		} else if iq.IsSet() {
			if !x.registered {
				// ...register a new user...
				x.registerNewUser(iq, q)
			} else {
				// return a <not-acceptable/> stanza error if an entity attempts to register a second identity
				x.strm.SendElement(iq.NotAcceptableError())
			}
		} else {
			x.strm.SendElement(iq.BadRequestError())
		}
	} else if iq.IsSet() {
		if q.FindElement("remove") != nil {
			// remove user
			x.cancelRegistration(iq, q)
		} else {
			user := q.FindElement("username")
			password := q.FindElement("password")
			if user != nil && password != nil {
				// change password
				x.changePassword(password.Text(), user.Text(), iq)
			} else {
				x.strm.SendElement(iq.BadRequestError())
			}
		}
	} else {
		x.strm.SendElement(iq.BadRequestError())
	}
}

func (x *XEPRegister) sendRegistrationFields(iq *xml.IQ, query *xml.Element) {
	if query.ElementsCount() > 0 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	result := iq.ResultIQ()
	q := xml.NewMutableElementNamespace("query", registerNamespace)
	q.AppendElement(xml.NewElementName("username"))
	q.AppendElement(xml.NewElementName("password"))
	result.AppendElement(q.Copy())
	x.strm.SendElement(result)
}

func (x *XEPRegister) registerNewUser(iq *xml.IQ, query *xml.Element) {
}

func (x *XEPRegister) cancelRegistration(iq *xml.IQ, query *xml.Element) {
}

func (x *XEPRegister) changePassword(password string, user string, iq *xml.IQ) {
}

func (x *XEPRegister) isValidToJid(jid *xml.JID) bool {
	if x.strm.Authenticated() {
		return jid.IsServer()
	} else {
		return jid.IsServer() || (jid.IsBare() && jid.Node() == x.strm.Username())
	}
}
