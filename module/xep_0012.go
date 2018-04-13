/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"strconv"
	"time"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const lastActivityNamespace = "jabber:iq:last"

type XEPLastActivity struct {
	stm       c2s.Stream
	startTime time.Time
}

// NewXEPLastActivity returns a last activity IQ handler module.
func NewXEPLastActivity(stm c2s.Stream) *XEPLastActivity {
	return &XEPLastActivity{
		stm:       stm,
		startTime: time.Now(),
	}
}

// AssociatedNamespaces returns namespaces associated
// with private storage module.
func (x *XEPLastActivity) AssociatedNamespaces() []string {
	return []string{lastActivityNamespace}
}

// Done signals stream termination.
func (x *XEPLastActivity) Done() {
}

// MatchesIQ returns whether or not an IQ should be
// processed by the last activity module.
func (x *XEPLastActivity) MatchesIQ(iq *xml.IQ) bool {
	return iq.IsGet() && iq.Elements().ChildNamespace("query", lastActivityNamespace) != nil
}

// ProcessIQ processes a private storage IQ taking according actions
// over the associated stream.
func (x *XEPLastActivity) ProcessIQ(iq *xml.IQ) {
	toJID := iq.ToJID()
	if toJID.IsServer() {
		x.sendServerUptime(iq)
	} else if toJID.IsBare() {
		ri, err := storage.Instance().FetchRosterItem(x.stm.Username(), toJID.Node())
		if err != nil {
			log.Error(err)
			x.stm.SendElement(iq.InternalServerError())
			return
		}
		if ri != nil {
			switch ri.Subscription {
			case subscriptionTo, subscriptionBoth:
				x.sendUserLastActivity(iq, toJID.Node())
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

func (x *XEPLastActivity) sendUserLastActivity(iq *xml.IQ, username string) {
	if len(c2s.Instance().AvailableStreams(username)) > 0 { // user online
		x.sendReply(iq, 0, "")
		return
	}
	usr, err := storage.Instance().FetchUser(username)
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
