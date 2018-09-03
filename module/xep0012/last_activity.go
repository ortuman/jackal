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
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 2048

const lastActivityNamespace = "jabber:iq:last"

// LastActivity represents a last activity stream module.
type LastActivity struct {
	startTime  time.Time
	actorCh    chan func()
	shutdownCh <-chan struct{}
}

// New returns a last activity IQ handler module.
func New(disco *xep0030.DiscoInfo, shutdownCh <-chan struct{}) *LastActivity {
	x := &LastActivity{
		startTime:  time.Now(),
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: shutdownCh,
	}
	go x.loop()
	if disco != nil {
		disco.RegisterServerFeature(lastActivityNamespace)
		disco.RegisterAccountFeature(lastActivityNamespace)
	}
	return x
}

// MatchesIQ returns whether or not an IQ should be
// processed by the last activity module.
func (x *LastActivity) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", lastActivityNamespace) != nil
}

// ProcessIQ processes a last activity IQ taking
// according actions over the associated stream.
func (x *LastActivity) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	x.actorCh <- func() { x.processIQ(iq, stm) }
}

// runs on it's own goroutine
func (x *LastActivity) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.shutdownCh:
			return
		}
	}
}

func (x *LastActivity) processIQ(iq *xmpp.IQ, stm stream.C2S) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if toJID.IsServer() {
		x.sendServerUptime(iq, stm)
	} else if toJID.IsBare() {
		ok, err := x.isSubscribedTo(toJID, fromJID)
		if err != nil {
			log.Error(err)
			stm.SendElement(iq.InternalServerError())
			return
		}
		if ok {
			x.sendUserLastActivity(iq, toJID, stm)
		} else {
			stm.SendElement(iq.ForbiddenError())
		}
	} else {
		stm.SendElement(iq.BadRequestError())
	}
}

func (x *LastActivity) sendServerUptime(iq *xmpp.IQ, stm stream.C2S) {
	secs := int(time.Duration(time.Now().UnixNano()-x.startTime.UnixNano()) / time.Second)
	x.sendReply(iq, secs, "", stm)
}

func (x *LastActivity) sendUserLastActivity(iq *xmpp.IQ, to *jid.JID, stm stream.C2S) {
	if len(router.UserStreams(to.Node())) > 0 { // user is online
		x.sendReply(iq, 0, "", stm)
		return
	}
	usr, err := storage.Instance().FetchUser(to.Node())
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	if usr == nil {
		stm.SendElement(iq.ItemNotFoundError())
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
	x.sendReply(iq, secs, status, stm)
}

func (x *LastActivity) sendReply(iq *xmpp.IQ, secs int, status string, stm stream.C2S) {
	q := xmpp.NewElementNamespace("query", lastActivityNamespace)
	q.SetText(status)
	q.SetAttribute("seconds", strconv.Itoa(secs))
	res := iq.ResultIQ()
	res.AppendElement(q)
	stm.SendElement(res)
}

func (x *LastActivity) isSubscribedTo(contact *jid.JID, userJID *jid.JID) (bool, error) {
	if contact.Matches(userJID, jid.MatchesBare) {
		return true, nil
	}
	ri, err := storage.Instance().FetchRosterItem(userJID.Node(), contact.ToBareJID().String())
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
