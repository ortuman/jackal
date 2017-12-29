/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/entity"
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
	userEl := query.FindElement("username")
	passwordEl := query.FindElement("password")
	if userEl == nil || passwordEl == nil || len(userEl.Text()) == 0 || len(passwordEl.Text()) == 0 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	exists, err := storage.Instance().UserExists(userEl.Text())
	if err != nil {
		log.Errorf("%v", err)
		x.strm.SendElement(iq.InternalServerError())
		return
	}
	if exists {
		x.strm.SendElement(iq.ConflictError())
		return
	}
	user := entity.User{
		Username: userEl.Text(),
		Password: passwordEl.Text(),
	}
	if err := storage.Instance().InsertOrUpdateUser(&user); err != nil {
		log.Errorf("%v", err)
		x.strm.SendElement(iq.InternalServerError())
		return
	}
	x.strm.SendElement(iq.ResultIQ())
	x.registered = true
}

func (x *XEPRegister) cancelRegistration(iq *xml.IQ, query *xml.Element) {
	if !x.cfg.AllowCancel {
		x.strm.SendElement(iq.NotAllowedError())
		return
	}
	if query.ElementsCount() != 1 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	if err := storage.Instance().DeleteUser(x.strm.Username()); err != nil {
		log.Error(err)
		x.strm.SendElement(iq.InternalServerError())
		return
	}
	x.strm.SendElement(iq.ResultIQ())
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
