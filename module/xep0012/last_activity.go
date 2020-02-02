/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0012

import (
	"context"
	"strconv"
	"time"

	"github.com/ortuman/jackal/log"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const lastActivityNamespace = "jabber:iq:last"

// LastActivity represents a last activity stream module.
type LastActivity struct {
	router    router.GlobalRouter
	userRep   repository.User
	rosterRep repository.Roster
	startTime time.Time
	runQueue  *runqueue.RunQueue
}

// New returns a last activity IQ handler module.
func New(disco *xep0030.DiscoInfo, router router.GlobalRouter, userRep repository.User, rosterRep repository.Roster) *LastActivity {
	x := &LastActivity{
		runQueue:  runqueue.New("xep0012"),
		router:    router,
		userRep:   userRep,
		rosterRep: rosterRep,
		startTime: time.Now(),
	}
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
func (x *LastActivity) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq)
	})
}

// Shutdown shuts down last activity module.
func (x *LastActivity) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *LastActivity) processIQ(ctx context.Context, iq *xmpp.IQ) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if toJID.IsServer() {
		x.sendServerUptime(ctx, iq)
	} else if toJID.IsBare() {
		ok, err := x.isSubscribedTo(ctx, toJID, fromJID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
		if ok {
			x.sendUserLastActivity(ctx, iq, toJID)
		} else {
			_ = x.router.Route(ctx, iq.ForbiddenError())
		}
	} else {
		_ = x.router.Route(ctx, iq.BadRequestError())
	}
}

func (x *LastActivity) sendServerUptime(ctx context.Context, iq *xmpp.IQ) {
	secs := int(time.Duration(time.Now().UnixNano()-x.startTime.UnixNano()) / time.Second)
	x.sendReply(ctx, iq, secs, "")
}

func (x *LastActivity) sendUserLastActivity(ctx context.Context, iq *xmpp.IQ, to *jid.JID) {
	if len(x.router.LocalStreams(to.Node())) > 0 { // user is online
		x.sendReply(ctx, iq, 0, "")
		return
	}
	usr, err := x.userRep.FetchUser(ctx, to.Node())
	if err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
		return
	}
	if usr == nil {
		_ = x.router.Route(ctx, iq.ItemNotFoundError())
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
	x.sendReply(ctx, iq, secs, status)
}

func (x *LastActivity) sendReply(ctx context.Context, iq *xmpp.IQ, secs int, status string) {
	q := xmpp.NewElementNamespace("query", lastActivityNamespace)
	q.SetText(status)
	q.SetAttribute("seconds", strconv.Itoa(secs))
	res := iq.ResultIQ()
	res.AppendElement(q)
	_ = x.router.Route(ctx, res)
}

func (x *LastActivity) isSubscribedTo(ctx context.Context, contact *jid.JID, userJID *jid.JID) (bool, error) {
	if contact.MatchesWithOptions(userJID, jid.MatchesBare) {
		return true, nil
	}
	ri, err := x.rosterRep.FetchRosterItem(ctx, userJID.Node(), contact.ToBareJID().String())
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
