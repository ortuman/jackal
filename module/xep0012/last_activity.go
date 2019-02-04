/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0012

import (
	"strconv"
	"time"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 2048

const lastActivityNamespace = "jabber:iq:last"

// LastActivity represents a last activity stream module.
type LastActivity struct {
	router     *router.Router
	startTime  time.Time
	actorCh    chan func()
	shutdownCh chan chan error
}

// New returns a last activity IQ handler module.
func New(disco *xep0030.DiscoInfo, router *router.Router) *LastActivity {
	x := &LastActivity{
		router:     router,
		startTime:  time.Now(),
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan error),
	}
	go x.loop()
	if disco != nil {
		disco.RegisterServerFeature(lastActivityNamespace)
		disco.RegisterAccountFeature(lastActivityNamespace)
	}
	return x
}

// MatchesIQ returns whether or not an IQ should be processed by the last activity module.
func (x *LastActivity) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", lastActivityNamespace) != nil
}

// ProcessIQ processes a last activity IQ taking according actions over the associated stream.
func (x *LastActivity) ProcessIQ(iq *xmpp.IQ) {
	x.actorCh <- func() {
		x.processIQ(iq)
	}
}

// Shutdown shuts down last activity module.
func (x *LastActivity) Shutdown() error {
	c := make(chan error)
	x.shutdownCh <- c
	return <-c
}

// runs on it's own goroutine
func (x *LastActivity) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case c := <-x.shutdownCh:
			c <- nil
			return
		}
	}
}

func (x *LastActivity) processIQ(iq *xmpp.IQ) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if toJID.IsServer() {
		x.sendServerUptime(iq)
	} else if toJID.IsBare() {
		ok, err := x.isSubscribedTo(toJID, fromJID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
		if ok {
			x.sendUserLastActivity(iq, toJID)
		} else {
			_ = x.router.Route(iq.ForbiddenError())
		}
	} else {
		_ = x.router.Route(iq.BadRequestError())
	}
}

func (x *LastActivity) sendServerUptime(iq *xmpp.IQ) {
	secs := int(time.Duration(time.Now().UnixNano()-x.startTime.UnixNano()) / time.Second)
	x.sendReply(iq, secs, "")
}

func (x *LastActivity) sendUserLastActivity(iq *xmpp.IQ, to *jid.JID) {
	if len(x.router.UserStreams(to.Node())) > 0 { // user is online
		x.sendReply(iq, 0, "")
		return
	}
	usr, err := storage.FetchUser(to.Node())
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	if usr == nil {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	var secs int
	var status string
	if p := usr.LastPresence; p != nil {
		secs = int(time.Duration(time.Now().UnixNano()-usr.LastPresenceAt.UnixNano()) / time.Second)
		if st := p.Elements().Child("status"); st != nil {
			status = st.Text()
		}
	}
	x.sendReply(iq, secs, status)
}

func (x *LastActivity) sendReply(iq *xmpp.IQ, secs int, status string) {
	q := xmpp.NewElementNamespace("query", lastActivityNamespace)
	q.SetText(status)
	q.SetAttribute("seconds", strconv.Itoa(secs))
	res := iq.ResultIQ()
	res.AppendElement(q)
	_ = x.router.Route(res)
}

func (x *LastActivity) isSubscribedTo(contact *jid.JID, userJID *jid.JID) (bool, error) {
	if contact.Matches(userJID, jid.MatchesBare) {
		return true, nil
	}
	ri, err := storage.FetchRosterItem(userJID.Node(), contact.ToBareJID().String())
	if err != nil {
		return false, err
	}
	if ri == nil {
		return false, nil
	}
	switch ri.Subscription {
	case rostermodel.SubscriptionTo, rostermodel.SubscriptionBoth:
		return true, nil
	default:
		return false, nil
	}
}
