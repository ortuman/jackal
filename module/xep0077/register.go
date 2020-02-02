/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0077

import (
	"context"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

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
	cfg      *Config
	router   router.GlobalRouter
	runQueue *runqueue.RunQueue
	rep      repository.User
}

// New returns an in-band registration IQ handler.
func New(config *Config, disco *xep0030.DiscoInfo, router router.GlobalRouter, userRep repository.User) *Register {
	r := &Register{
		cfg:      config,
		router:   router,
		runQueue: runqueue.New("xep0077"),
		rep:      userRep,
	}
	if disco != nil {
		disco.RegisterServerFeature(registerNamespace)
	}
	return r
}

// MatchesIQ returns whether or not an IQ should be processed by the in-band registration module.
func (x *Register) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.Elements().ChildNamespace("query", registerNamespace) != nil
}

// ProcessIQ processes an in-band registration IQ taking according actions over the associated stream.
func (x *Register) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		if stm := x.router.LocalStream(iq.FromJID().Node(), iq.FromJID().Resource()); stm != nil {
			x.processIQ(ctx, iq, stm)
		}
	})
}

// ProcessIQWithStream processes an in-band registration IQ taking according actions over a referenced stream.
func (x *Register) ProcessIQWithStream(ctx context.Context, iq *xmpp.IQ, stm stream.C2S) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq, stm)
	})
}

// Shutdown shuts down in-band registration module.
func (x *Register) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *Register) processIQ(ctx context.Context, iq *xmpp.IQ, stm stream.C2S) {
	if !x.isValidToJid(iq.ToJID(), stm) {
		stm.SendElement(ctx, iq.ForbiddenError())
		return
	}
	q := iq.Elements().ChildNamespace("query", registerNamespace)
	if !stm.IsAuthenticated() {
		if iq.IsGet() {
			if !x.cfg.AllowRegistration {
				stm.SendElement(ctx, iq.NotAllowedError())
				return
			}
			// ...send registration fields to requester entity...
			x.sendRegistrationFields(ctx, iq, q, stm)
		} else if iq.IsSet() {
			registered, _ := stm.Value(xep077RegisteredCtxKey).(bool)
			if !registered {
				// ...register a new user...
				x.registerNewUser(ctx, iq, q, stm)
			} else {
				// return a <not-acceptable/> stanza error if an entity attempts to register a second identity
				stm.SendElement(ctx, iq.NotAcceptableError())
			}
		} else {
			stm.SendElement(ctx, iq.BadRequestError())
		}
	} else if iq.IsSet() {
		if q.Elements().Child("remove") != nil {
			// remove user
			x.cancelRegistration(ctx, iq, q, stm)
		} else {
			user := q.Elements().Child("username")
			password := q.Elements().Child("password")
			if user != nil && password != nil {
				// change password
				x.changePassword(ctx, password.Text(), user.Text(), iq, stm)
			} else {
				stm.SendElement(ctx, iq.BadRequestError())
			}
		}
	} else {
		stm.SendElement(ctx, iq.BadRequestError())
	}
}

func (x *Register) sendRegistrationFields(ctx context.Context, iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) {
	if query.Elements().Count() > 0 {
		stm.SendElement(ctx, iq.BadRequestError())
		return
	}
	result := iq.ResultIQ()
	q := xmpp.NewElementNamespace("query", registerNamespace)
	q.AppendElement(xmpp.NewElementName("username"))
	q.AppendElement(xmpp.NewElementName("password"))
	result.AppendElement(q)
	stm.SendElement(ctx, result)
}

func (x *Register) registerNewUser(ctx context.Context, iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) {
	userEl := query.Elements().Child("username")
	passwordEl := query.Elements().Child("password")
	if userEl == nil || passwordEl == nil || len(userEl.Text()) == 0 || len(passwordEl.Text()) == 0 {
		stm.SendElement(ctx, iq.BadRequestError())
		return
	}
	exists, err := x.rep.UserExists(ctx, userEl.Text())
	if err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.InternalServerError())
		return
	}
	if exists {
		stm.SendElement(ctx, iq.ConflictError())
		return
	}
	user := model.User{
		Username:     userEl.Text(),
		Password:     passwordEl.Text(),
		LastPresence: xmpp.NewPresence(stm.JID(), stm.JID(), xmpp.UnavailableType),
	}
	if err := x.rep.UpsertUser(ctx, &user); err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.InternalServerError())
		return
	}
	stm.SendElement(ctx, iq.ResultIQ())
	stm.SetValue(xep077RegisteredCtxKey, true) // mark as registered
}

func (x *Register) cancelRegistration(ctx context.Context, iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) {
	if !x.cfg.AllowCancel {
		stm.SendElement(ctx, iq.NotAllowedError())
		return
	}
	if query.Elements().Count() > 1 {
		stm.SendElement(ctx, iq.BadRequestError())
		return
	}
	if err := x.rep.DeleteUser(ctx, stm.Username()); err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.InternalServerError())
		return
	}
	stm.SendElement(ctx, iq.ResultIQ())
}

func (x *Register) changePassword(ctx context.Context, password string, username string, iq *xmpp.IQ, stm stream.C2S) {
	if !x.cfg.AllowChange {
		stm.SendElement(ctx, iq.NotAllowedError())
		return
	}
	if username != stm.Username() {
		stm.SendElement(ctx, iq.NotAllowedError())
		return
	}
	if !stm.IsSecured() {
		// channel isn't safe enough to enable a password change
		stm.SendElement(ctx, iq.NotAuthorizedError())
		return
	}
	user, err := x.rep.FetchUser(ctx, username)
	if err != nil {
		log.Error(err)
		stm.SendElement(ctx, iq.InternalServerError())
		return
	}
	if user == nil {
		stm.SendElement(ctx, iq.ResultIQ())
		return
	}
	if user.Password != password {
		user.Password = password
		if err := x.rep.UpsertUser(ctx, user); err != nil {
			log.Error(err)
			stm.SendElement(ctx, iq.InternalServerError())
			return
		}
	}
	stm.SendElement(ctx, iq.ResultIQ())
}

func (x *Register) isValidToJid(j *jid.JID, stm stream.C2S) bool {
	if stm.IsAuthenticated() && (j.IsBare() && j.Node() != stm.Username()) {
		return false
	}
	return true
}
