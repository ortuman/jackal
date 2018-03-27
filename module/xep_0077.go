/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const registerNamespace = "jabber:iq:register"

// XEPRegister represents an in-band server stream module.
type XEPRegister struct {
	cfg        *config.ModRegistration
	strm       c2s.Stream
	registered bool
}

// NewXEPRegister returns an in-band registration IQ handler.
func NewXEPRegister(config *config.ModRegistration, strm c2s.Stream) *XEPRegister {
	return &XEPRegister{
		cfg:  config,
		strm: strm,
	}
}

// AssociatedNamespaces returns namespaces associated
// with in-band registration module.
func (x *XEPRegister) AssociatedNamespaces() []string {
	return []string{registerNamespace}
}

// Done signals stream termination.
func (x *XEPRegister) Done() {
}

// MatchesIQ returns whether or not an IQ should be
// processed by the in-band registration module.
func (x *XEPRegister) MatchesIQ(iq *xml.IQ) bool {
	return iq.Elements().ChildNamespace("query", registerNamespace) != nil
}

// ProcessIQ processes an in-band registration IQ
// taking according actions over the associated stream.
func (x *XEPRegister) ProcessIQ(iq *xml.IQ) {
	if !x.isValidToJid(iq.ToJID()) {
		x.strm.SendElement(iq.ForbiddenError())
		return
	}

	q := iq.Elements().ChildNamespace("query", registerNamespace)
	if !x.strm.IsAuthenticated() {
		if iq.IsGet() {
			if !x.cfg.AllowRegistration {
				x.strm.SendElement(iq.NotAllowedError())
				return
			}
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
		if q.Elements().Child("remove") != nil {
			// remove user
			x.cancelRegistration(iq, q)
		} else {
			user := q.Elements().Child("username")
			password := q.Elements().Child("password")
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

func (x *XEPRegister) sendRegistrationFields(iq *xml.IQ, query xml.ElementNode) {
	if query.Elements().Count() > 0 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	result := iq.ResultIQ()
	q := xml.NewElementNamespace("query", registerNamespace)
	q.AppendElement(xml.NewElementName("username"))
	q.AppendElement(xml.NewElementName("password"))
	result.AppendElement(q)
	x.strm.SendElement(result)
}

func (x *XEPRegister) registerNewUser(iq *xml.IQ, query xml.ElementNode) {
	userEl := query.Elements().Child("username")
	passwordEl := query.Elements().Child("password")
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
	user := model.User{
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

func (x *XEPRegister) cancelRegistration(iq *xml.IQ, query xml.ElementNode) {
	if !x.cfg.AllowCancel {
		x.strm.SendElement(iq.NotAllowedError())
		return
	}
	if query.Elements().Count() > 1 {
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

func (x *XEPRegister) changePassword(password string, username string, iq *xml.IQ) {
	if !x.cfg.AllowChange {
		x.strm.SendElement(iq.NotAllowedError())
		return
	}
	if username != x.strm.Username() {
		x.strm.SendElement(iq.NotAllowedError())
		return
	}
	if !x.strm.IsSecured() {
		// channel isn't safe enough to enable a password change
		x.strm.SendElement(iq.NotAuthorizedError())
		return
	}
	user, err := storage.Instance().FetchUser(username)
	if err != nil {
		log.Error(err)
		x.strm.SendElement(iq.InternalServerError())
		return
	}
	if user == nil {
		x.strm.SendElement(iq.ResultIQ())
		return
	}
	if user.Password != password {
		user.Password = password
		if err := storage.Instance().InsertOrUpdateUser(user); err != nil {
			log.Error(err)
			x.strm.SendElement(iq.InternalServerError())
			return
		}
	}
	x.strm.SendElement(iq.ResultIQ())
}

func (x *XEPRegister) isValidToJid(jid *xml.JID) bool {
	if x.strm.IsAuthenticated() {
		return jid.IsServer()
	}
	return jid.IsServer() || (jid.IsBare() && jid.Node() == x.strm.Username())
}
