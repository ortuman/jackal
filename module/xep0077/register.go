/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0077

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 2048

const registerNamespace = "jabber:iq:register"

const xep077RegisteredCtxKey = "xep0077:registered"

// Config represents XMPP In-Band Registration module (XEP-0077) configuration.
type Config struct {
	AllowRegistration bool `yaml:"allow_registration"`
	AllowChange       bool `yaml:"allow_change"`
	AllowCancel       bool `yaml:"allow_cancel"`
}

// Register represents an in-band server stream module.
type Register struct {
	cfg        *Config
	actorCh    chan func()
	shutdownCh chan chan bool
}

// New returns an in-band registration IQ handler.
func New(config *Config, disco *xep0030.DiscoInfo) (*Register, chan<- chan bool) {
	r := &Register{
		cfg:        config,
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan bool),
	}
	go r.loop()
	if disco != nil {
		disco.RegisterServerFeature(registerNamespace)
	}
	return r, r.shutdownCh
}

// MatchesIQ returns whether or not an IQ should be
// processed by the in-band registration module.
func (x *Register) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.Elements().ChildNamespace("query", registerNamespace) != nil
}

// ProcessIQ processes an in-band registration IQ
// taking according actions over the associated stream.
func (x *Register) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	x.actorCh <- func() { x.processIQ(iq, stm) }
}

// runs on it's own goroutine
func (x *Register) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case c := <-x.shutdownCh:
			c <- true
			return
		}
	}
}

func (x *Register) processIQ(iq *xmpp.IQ, stm stream.C2S) {
	if !x.isValidToJid(iq.ToJID(), stm) {
		stm.SendElement(iq.ForbiddenError())
		return
	}
	q := iq.Elements().ChildNamespace("query", registerNamespace)
	if !stm.IsAuthenticated() {
		if iq.IsGet() {
			if !x.cfg.AllowRegistration {
				stm.SendElement(iq.NotAllowedError())
				return
			}
			// ...send registration fields to requester entity...
			x.sendRegistrationFields(iq, q, stm)
		} else if iq.IsSet() {
			if !stm.GetBool(xep077RegisteredCtxKey) {
				// ...register a new user...
				x.registerNewUser(iq, q, stm)
			} else {
				// return a <not-acceptable/> stanza error if an entity attempts to register a second identity
				stm.SendElement(iq.NotAcceptableError())
			}
		} else {
			stm.SendElement(iq.BadRequestError())
		}
	} else if iq.IsSet() {
		if q.Elements().Child("remove") != nil {
			// remove user
			x.cancelRegistration(iq, q, stm)
		} else {
			user := q.Elements().Child("username")
			password := q.Elements().Child("password")
			if user != nil && password != nil {
				// change password
				x.changePassword(password.Text(), user.Text(), iq, stm)
			} else {
				stm.SendElement(iq.BadRequestError())
			}
		}
	} else {
		stm.SendElement(iq.BadRequestError())
	}
}

func (x *Register) sendRegistrationFields(iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) {
	if query.Elements().Count() > 0 {
		stm.SendElement(iq.BadRequestError())
		return
	}
	result := iq.ResultIQ()
	q := xmpp.NewElementNamespace("query", registerNamespace)
	q.AppendElement(xmpp.NewElementName("username"))
	q.AppendElement(xmpp.NewElementName("password"))
	result.AppendElement(q)
	stm.SendElement(result)
}

func (x *Register) registerNewUser(iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) {
	userEl := query.Elements().Child("username")
	passwordEl := query.Elements().Child("password")
	if userEl == nil || passwordEl == nil || len(userEl.Text()) == 0 || len(passwordEl.Text()) == 0 {
		stm.SendElement(iq.BadRequestError())
		return
	}
	exists, err := storage.UserExists(userEl.Text())
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	if exists {
		stm.SendElement(iq.ConflictError())
		return
	}
	user := model.User{
		Username:     userEl.Text(),
		Password:     passwordEl.Text(),
		LastPresence: xmpp.NewPresence(stm.JID(), stm.JID(), xmpp.UnavailableType),
	}
	if err := storage.InsertOrUpdateUser(&user); err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	stm.SendElement(iq.ResultIQ())
	stm.SetBool(xep077RegisteredCtxKey, true) // mark as registered
}

func (x *Register) cancelRegistration(iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) {
	if !x.cfg.AllowCancel {
		stm.SendElement(iq.NotAllowedError())
		return
	}
	if query.Elements().Count() > 1 {
		stm.SendElement(iq.BadRequestError())
		return
	}
	if err := storage.DeleteUser(stm.Username()); err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	stm.SendElement(iq.ResultIQ())
}

func (x *Register) changePassword(password string, username string, iq *xmpp.IQ, stm stream.C2S) {
	if !x.cfg.AllowChange {
		stm.SendElement(iq.NotAllowedError())
		return
	}
	if username != stm.Username() {
		stm.SendElement(iq.NotAllowedError())
		return
	}
	if !stm.IsSecured() {
		// channel isn't safe enough to enable a password change
		stm.SendElement(iq.NotAuthorizedError())
		return
	}
	user, err := storage.FetchUser(username)
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	if user == nil {
		stm.SendElement(iq.ResultIQ())
		return
	}
	if user.Password != password {
		user.Password = password
		if err := storage.InsertOrUpdateUser(user); err != nil {
			log.Error(err)
			stm.SendElement(iq.InternalServerError())
			return
		}
	}
	stm.SendElement(iq.ResultIQ())
}

func (x *Register) isValidToJid(j *jid.JID, stm stream.C2S) bool {
	if stm.IsAuthenticated() && (j.IsBare() && j.Node() != stm.Username()) {
		return false
	}
	return true
}
