/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0012

import (
	"strconv"
	"time"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const lastActivityNamespace = "jabber:iq:last"

// XEPLastActivity represents a last activity stream module.
type XEPLastActivity struct {
	stm       c2s.Stream
	startTime time.Time
}

// New returns a last activity IQ handler module.
func New(stm c2s.Stream) *XEPLastActivity {
	return &XEPLastActivity{
		stm:       stm,
		startTime: time.Now(),
	}
}

// AssociatedNamespaces returns namespaces associated
// with last activity module.
func (x *XEPLastActivity) AssociatedNamespaces() []string {
	return []string{lastActivityNamespace}
}

// MatchesIQ returns whether or not an IQ should be
// processed by the last activity module.
func (x *XEPLastActivity) MatchesIQ(iq *xml.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", lastActivityNamespace) != nil
}

// ProcessIQ processes a last activity IQ taking according actions
// over the associated stream.
func (x *XEPLastActivity) ProcessIQ(iq *xml.IQ) {
	toJID := iq.ToJID()
	if toJID.IsServer() {
		x.sendServerUptime(iq)
	} else if toJID.IsBare() {
		ri, err := storage.Instance().FetchRosterItem(x.stm.Username(), toJID.ToBareJID().String())
		if err != nil {
			log.Error(err)
			x.stm.SendElement(iq.InternalServerError())
			return
		}
		if ri != nil {
			switch ri.Subscription {
			case roster.SubscriptionTo, roster.SubscriptionBoth:
				x.sendUserLastActivity(iq, toJID)
			default:
				x.stm.SendElement(iq.ForbiddenError())
			}
		} else {
			x.stm.SendElement(iq.ForbiddenError())
		}
	}
}

func (x *XEPLastActivity) sendServerUptime(iq *xml.IQ) {
	secs := int(time.Duration(time.Now().UnixNano()-x.startTime.UnixNano()) / time.Second)
	x.sendReply(iq, secs, "")
}

func (x *XEPLastActivity) sendUserLastActivity(iq *xml.IQ, to *xml.JID) {
	if len(c2s.Instance().StreamsMatchingJID(to.ToBareJID())) > 0 { // user online
		x.sendReply(iq, 0, "")
		return
	}
	usr, err := storage.Instance().FetchUser(to.Node())
	if err != nil {
		log.Error(err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	if usr == nil {
		x.stm.SendElement(iq.ItemNotFoundError())
		return
	}
	secs := int(time.Duration(time.Now().UnixNano()-usr.LoggedOutAt.UnixNano()) / time.Second)
	x.sendReply(iq, secs, usr.LoggedOutStatus)
}

func (x *XEPLastActivity) sendReply(iq *xml.IQ, secs int, status string) {
	q := xml.NewElementNamespace("query", lastActivityNamespace)
	q.SetText(status)
	q.SetAttribute("seconds", strconv.Itoa(secs))
	res := iq.ResultIQ()
	res.AppendElement(q)
	x.stm.SendElement(res)
}
