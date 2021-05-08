// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package roster

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/c2s"
	"github.com/ortuman/jackal/pkg/event"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/repository"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const (
	requestedInfoKey = "roster:requested"
	availableInfoKey = "roster:available"

	rosterNamespace = "jabber:iq:roster"
)

const (
	// ModuleName represents roster module name.
	ModuleName = "roster"
)

// Roster represents a roster module type.
type Roster struct {
	rep    repository.Repository
	resMng resourceManager
	router router.Router
	hosts  hosts
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New returns a new initialized Roster instance.
func New(
	router router.Router,
	rep repository.Repository,
	resMng *c2s.ResourceManager,
	hosts *host.Hosts,
	sonar *sonar.Sonar,
) *Roster {
	return &Roster{
		router: router,
		rep:    rep,
		resMng: resMng,
		hosts:  hosts,
		sn:     sonar,
	}
}

// Name returns roster module name.
func (r *Roster) Name() string { return ModuleName }

// StreamFeature returns roster stream feature.
func (r *Roster) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return stravaganza.NewBuilder("ver").
		WithAttribute(stravaganza.Namespace, "urn:xmpp:features:rosterver").
		Build(), nil
}

// ServerFeatures returns roster server disco features.
func (r *Roster) ServerFeatures(_ context.Context) ([]string, error) { return nil, nil }

// AccountFeatures returns roster account disco features.
func (r *Roster) AccountFeatures(_ context.Context) ([]string, error) { return nil, nil }

// MatchesNamespace tells whether namespace matches roster module.
func (r *Roster) MatchesNamespace(namespace string, serverTarget bool) bool {
	if serverTarget {
		return false
	}
	return namespace == rosterNamespace
}

// ProcessIQ process a roster iq.
func (r *Roster) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet():
		return r.sendRoster(ctx, iq)
	case iq.IsSet():
		return r.updateRoster(ctx, iq)
	}
	return nil
}

// Start starts roster module.
func (r *Roster) Start(_ context.Context) error {
	r.subs = append(r.subs, r.sn.Subscribe(event.C2SStreamPresenceReceived, r.onPresenceRecv))
	r.subs = append(r.subs, r.sn.Subscribe(event.S2SInStreamPresenceReceived, r.onPresenceRecv))
	r.subs = append(r.subs, r.sn.Subscribe(event.UserDeleted, r.onUserDeleted))

	log.Infow("Started roster module", "xep", "roster")
	return nil
}

// Stop stops roster module.
func (r *Roster) Stop(_ context.Context) error {
	for _, sub := range r.subs {
		r.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped roster module", "xep", "roster")
	return nil
}

func (r *Roster) onPresenceRecv(ctx context.Context, ev sonar.Event) error {
	var pr *stravaganza.Presence
	switch inf := ev.Info().(type) {
	case *event.C2SStreamEventInfo:
		pr, _ = inf.Element.(*stravaganza.Presence)
	case *event.S2SStreamEventInfo:
		pr, _ = inf.Element.(*stravaganza.Presence)
	default:
		return nil
	}
	if pr.ToJID().IsFull() {
		return nil
	}
	if err := r.processPresence(ctx, pr); err != nil {
		return fmt.Errorf("roster: failed to process C2S presence: %s", err)
	}
	return nil
}

func (r *Roster) onUserDeleted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.UserEventInfo)
	return r.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if err := tx.DeleteRosterNotifications(ctx, inf.Username); err != nil {
			return err
		}
		return tx.DeleteRosterItems(ctx, inf.Username)
	})
}

func (r *Roster) processPresence(ctx context.Context, pr *stravaganza.Presence) error {
	switch pr.Attribute(stravaganza.Type) {
	case stravaganza.SubscribeType:
		return r.processSubscribe(ctx, pr)
	case stravaganza.SubscribedType:
		return r.processSubscribed(ctx, pr)
	case stravaganza.UnsubscribeType:
		return r.processUnsubscribe(ctx, pr)
	case stravaganza.UnsubscribedType:
		return r.processUnsubscribed(ctx, pr)
	case stravaganza.ProbeType:
		return r.processProbe(ctx, pr)
	case stravaganza.AvailableType, stravaganza.UnavailableType:
		return r.processAvailability(ctx, pr)
	}
	return nil
}

func (r *Roster) sendRoster(ctx context.Context, iq *stravaganza.IQ) error {
	q := iq.ChildNamespace("query", rosterNamespace)
	if q == nil || q.ChildrenCount() > 0 {
		_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	usrJID := iq.FromJID()

	// check against current roster version
	ver, err := r.rep.FetchRosterVersion(ctx, usrJID.Node())
	if err != nil {
		_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// return empty response in case version matches...
	if ver > 0 && ver == parseVer(q.Attribute("ver")) {
		_, _ = r.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))
		err = r.postRosterEvent(ctx, event.RosterRequested, &event.RosterEventInfo{
			Username: usrJID.Node(),
		})
		if err != nil {
			return err
		}
		return r.setStreamValue(ctx, usrJID.Node(), usrJID.Resource(), requestedInfoKey, true)
	}
	// ...return whole roster otherwise
	items, err := r.rep.FetchRosterItems(ctx, usrJID.Node())
	if err != nil {
		_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	sb := stravaganza.NewBuilder("query").
		WithAttribute(stravaganza.Namespace, rosterNamespace)
	for _, item := range items {
		sb.WithChild(encodeRosterItem(&item))
	}
	queryEl := sb.Build()

	// route roster
	_, _ = r.router.Route(ctx, xmpputil.MakeResultIQ(iq, queryEl))

	log.Infow("Fetched user roster", "jid", usrJID.String(), "xep", "roster")

	err = r.postRosterEvent(ctx, event.RosterRequested, &event.RosterEventInfo{
		Username: usrJID.Node(),
	})
	if err != nil {
		return err
	}
	return r.setStreamValue(ctx, usrJID.Node(), usrJID.Resource(), requestedInfoKey, true)
}

func (r *Roster) updateRoster(ctx context.Context, iq *stravaganza.IQ) error {
	q := iq.ChildNamespace("query", rosterNamespace)
	if q == nil {
		_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	items := q.Children("item")
	if len(items) != 1 {
		_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	ri, err := decodeRosterItem(items[0])
	if err != nil {
		_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	switch ri.Subscription {
	case rostermodel.Remove:
		if err := r.removeItem(ctx, ri, iq.FromJID().ToBareJID()); err != nil {
			_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
			return err
		}
	default:
		if err := r.updateItem(ctx, ri, iq.FromJID().Node()); err != nil {
			_, _ = r.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
			return err
		}
	}
	_, _ = r.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))
	return nil
}

func (r *Roster) processSubscribe(ctx context.Context, presence *stravaganza.Presence) error {
	userJID := presence.FromJID().ToBareJID()
	contactJID := presence.ToJID().ToBareJID()

	if r.hosts.IsLocalHost(userJID.Domain()) {
		usrRi, err := r.rep.FetchRosterItem(ctx, userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case rostermodel.To, rostermodel.Both:
				return nil // already subscribed...
			default:
				if !usrRi.Ask {
					usrRi.Ask = true
				} else {
					return nil // notification already sent...
				}
			}
		} else {
			// create roster item if not previously created
			usrRi = &rostermodel.Item{
				Username:     userJID.Node(),
				JID:          contactJID.String(),
				Subscription: rostermodel.None,
				Ask:          true,
			}
		}
		if err := r.upsertItem(ctx, usrRi); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "subscribe" with the user's bare JID as the 'from' address
	p := xmpputil.MakePresence(userJID, contactJID, stravaganza.SubscribeType, presence.AllChildren())

	if r.hosts.IsLocalHost(contactJID.Domain()) {
		// archive roster approval notification
		if err := r.upsertNotification(ctx, contactJID.Node(), userJID, p); err != nil {
			return err
		}
	}
	log.Infow("Processed 'subscribe' presence", "jid", contactJID, "username", userJID.Node(), "xep", "roster")

	_, _ = r.router.Route(ctx, p)
	return nil
}

func (r *Roster) processSubscribed(ctx context.Context, presence *stravaganza.Presence) error {
	userJID := presence.ToJID().ToBareJID()
	contactJID := presence.FromJID().ToBareJID()

	if r.hosts.IsLocalHost(contactJID.Domain()) {
		_, err := r.deleteNotification(ctx, contactJID.Node(), userJID)
		if err != nil {
			return err
		}
		cntRi, err := r.rep.FetchRosterItem(ctx, contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			switch cntRi.Subscription {
			case rostermodel.To:
				cntRi.Subscription = rostermodel.Both
			case rostermodel.None:
				cntRi.Subscription = rostermodel.From
			}
		} else {
			// create roster item if not previously created
			cntRi = &rostermodel.Item{
				Username:     contactJID.Node(),
				JID:          userJID.String(),
				Subscription: rostermodel.From,
				Ask:          false,
			}
		}
		if err := r.upsertItem(ctx, cntRi); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "subscribed" with the contact's bare JID as the 'from' address
	p := xmpputil.MakePresence(contactJID, userJID, stravaganza.SubscribedType, presence.AllChildren())

	if r.hosts.IsLocalHost(userJID.Domain()) {
		usrRi, err := r.rep.FetchRosterItem(ctx, userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case rostermodel.From:
				usrRi.Subscription = rostermodel.Both
			case rostermodel.None:
				usrRi.Subscription = rostermodel.To
			default:
				return nil
			}
			usrRi.Ask = false
			if err := r.upsertItem(ctx, usrRi); err != nil {
				return err
			}
		}
	}
	log.Infow("Processed 'subscribed' presence", "jid", contactJID, "username", userJID.Node(), "xep", "roster")

	_, _ = r.router.Route(ctx, p)
	return r.routePresencesFrom(ctx, contactJID.Node(), userJID, stravaganza.AvailableType)
}

func (r *Roster) processUnsubscribe(ctx context.Context, presence *stravaganza.Presence) error {
	userJID := presence.FromJID().ToBareJID()
	contactJID := presence.ToJID().ToBareJID()

	var usrSub string
	if r.hosts.IsLocalHost(userJID.Domain()) {
		usrRi, err := r.rep.FetchRosterItem(ctx, userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		usrSub = rostermodel.None
		if usrRi != nil {
			usrSub = usrRi.Subscription
			switch usrSub {
			case rostermodel.Both:
				usrRi.Subscription = rostermodel.From
			default:
				usrRi.Subscription = rostermodel.None
			}
			if err := r.upsertItem(ctx, usrRi); err != nil {
				return err
			}
		}
	}
	// stamp the presence stanza of type "unsubscribe" with the users's bare JID as the 'from' address
	p := xmpputil.MakePresence(userJID, contactJID, stravaganza.UnsubscribeType, presence.AllChildren())

	if r.hosts.IsLocalHost(contactJID.Domain()) {
		cntRi, err := r.rep.FetchRosterItem(ctx, contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			switch cntRi.Subscription {
			case rostermodel.Both:
				cntRi.Subscription = rostermodel.To
			default:
				cntRi.Subscription = rostermodel.None
			}
			if err := r.upsertItem(ctx, cntRi); err != nil {
				return err
			}
		}
	}
	_, _ = r.router.Route(ctx, p)

	if usrSub == rostermodel.To || usrSub == rostermodel.Both {
		if err := r.routePresencesFrom(ctx, contactJID.Node(), userJID, stravaganza.UnavailableType); err != nil {
			return err
		}
	}
	log.Infow("Processed 'unsubscribe' presence", "jid", contactJID, "username", userJID.Node(), "xep", "roster")
	return nil
}

func (r *Roster) processUnsubscribed(ctx context.Context, presence *stravaganza.Presence) error {
	userJID := presence.ToJID().ToBareJID()
	contactJID := presence.FromJID().ToBareJID()

	var cntSub string
	if r.hosts.IsLocalHost(contactJID.Domain()) {
		deleted, err := r.deleteNotification(ctx, contactJID.Node(), userJID)
		if err != nil {
			return err
		}
		// do not change subscription state if cancelling a subscription request
		if deleted {
			goto routePresence
		}
		cntRi, err := r.rep.FetchRosterItem(ctx, contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		cntSub = rostermodel.None
		if cntRi != nil {
			cntSub = cntRi.Subscription
			switch cntSub {
			case rostermodel.Both:
				cntRi.Subscription = rostermodel.To
			default:
				cntRi.Subscription = rostermodel.None
			}
			if err := r.upsertItem(ctx, cntRi); err != nil {
				return err
			}
		}
	}

routePresence:
	// stamp the presence stanza of type "unsubscribed" with the contact's bare JID as the 'from' address
	p := xmpputil.MakePresence(contactJID, userJID, stravaganza.UnsubscribedType, presence.AllChildren())

	if r.hosts.IsLocalHost(userJID.Domain()) {
		usrRi, err := r.rep.FetchRosterItem(ctx, userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			if !usrRi.Ask { // pending out...
				switch usrRi.Subscription {
				case rostermodel.Both:
					usrRi.Subscription = rostermodel.From
				default:
					usrRi.Subscription = rostermodel.None
				}
			}
			usrRi.Ask = false
			if err := r.upsertItem(ctx, usrRi); err != nil {
				return err
			}
		}
	}
	_, _ = r.router.Route(ctx, p)

	if cntSub == rostermodel.From || cntSub == rostermodel.Both {
		if err := r.routePresencesFrom(ctx, contactJID.Node(), userJID, stravaganza.UnavailableType); err != nil {
			return err
		}
	}
	log.Infow("Processed 'unsubscribed' presence", "jid", contactJID, "username", userJID.Node(), "xep", "roster")
	return nil
}

func (r *Roster) processProbe(ctx context.Context, presence *stravaganza.Presence) error {
	userJID := presence.FromJID().ToBareJID()
	contactJID := presence.ToJID().ToBareJID()

	if !r.hosts.IsLocalHost(contactJID.Domain()) {
		_, _ = r.router.Route(ctx, presence)
		return nil
	}
	ri, err := r.rep.FetchRosterItem(ctx, contactJID.Node(), userJID.String())
	if err != nil {
		return err
	}
	if ri == nil || (ri.Subscription != rostermodel.Both && ri.Subscription != rostermodel.From) {
		return nil // silently ignore
	}
	rss, err := r.resMng.GetResources(ctx, contactJID.Node())
	if err != nil {
		return err
	}
	for _, res := range rss {
		if !res.Presence.IsAvailable() {
			continue
		}
		p := xmpputil.MakePresence(res.JID, userJID, stravaganza.AvailableType, res.Presence.AllChildren())
		_, _ = r.router.Route(ctx, p)
	}
	log.Infow("Processed 'probe' presence", "jid", contactJID, "username", userJID.Node(), "xep", "roster")
	return nil
}

func (r *Roster) processAvailability(ctx context.Context, presence *stravaganza.Presence) error {
	fromJID := presence.FromJID()

	userJID := fromJID.ToBareJID()
	contactJID := presence.ToJID().ToBareJID()

	replyOnBehalf := r.hosts.IsLocalHost(userJID.Domain()) && userJID.MatchesWithOptions(contactJID, jid.MatchesBare)
	if !replyOnBehalf {
		_, _ = r.router.Route(ctx, presence)
		return nil
	}
	items, err := r.rep.FetchRosterItems(ctx, userJID.Node())
	if err != nil {
		return err
	}
	isAvailable := presence.IsAvailable()
	if isAvailable {
		sInf, err := r.getStreamInfo(fromJID.Node(), fromJID.Resource())
		if err != nil {
			return err
		}
		if sInf.Bool(availableInfoKey) {
			goto broadcastPresence
		}
		// send self-presence
		rss, err := r.resMng.GetResources(ctx, userJID.Node())
		if err != nil {
			return err
		}
		for _, res := range rss {
			pr := xmpputil.MakePresence(fromJID, res.JID, stravaganza.AvailableType, presence.AllChildren())
			_, _ = r.router.Route(ctx, pr)
		}
		// deliver pending notifications
		rns, err := r.rep.FetchRosterNotifications(ctx, userJID.Node())
		if err != nil {
			return err
		}
		for _, rn := range rns {
			_, _ = r.router.Route(ctx, rn.Presence)
		}
		// deliver roster presences
		for _, item := range items {
			switch item.Subscription {
			case rostermodel.To, rostermodel.Both:
				itemJID, _ := jid.NewWithString(item.JID, true)
				if r.hosts.IsLocalHost(itemJID.Domain()) {
					if err := r.routePresencesFrom(ctx, itemJID.Node(), fromJID, stravaganza.AvailableType); err != nil {
						return err
					}
					continue
				}
				// send probe presence to remote domain
				p := xmpputil.MakePresence(fromJID, itemJID, stravaganza.ProbeType, nil)
				_, _ = r.router.Route(ctx, p)
			}
		}
		// mark first avail
		if err := r.setStreamValue(ctx, fromJID.Node(), fromJID.Resource(), availableInfoKey, true); err != nil {
			return err
		}
	}

broadcastPresence:
	for _, item := range items {
		switch item.Subscription {
		case rostermodel.From, rostermodel.Both:
			itemJID, _ := jid.NewWithString(item.JID, true)
			p := xmpputil.MakePresence(presence.FromJID(), itemJID, presence.Type(), presence.AllChildren())
			_, _ = r.router.Route(ctx, p)
		}
	}
	if isAvailable {
		log.Infow("Processed 'available' presence", "jid", contactJID, "username", userJID.Node(), "xep", "roster")
	} else {
		log.Infow("Processed 'unavailable' presence", "jid", contactJID, "username", userJID.Node(), "xep", "roster")
	}
	return nil
}

func (r *Roster) updateItem(ctx context.Context, ri *rostermodel.Item, username string) error {
	usrRi, err := r.rep.FetchRosterItem(ctx, username, ri.JID)
	if err != nil {
		return err
	}
	if usrRi != nil {
		// update roster item
		if len(ri.Name) > 0 {
			usrRi.Name = ri.Name
		}
		usrRi.Groups = ri.Groups

	} else {
		usrRi = &rostermodel.Item{
			Username:     username,
			JID:          ri.JID,
			Name:         ri.Name,
			Subscription: rostermodel.None,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	if err := r.upsertItem(ctx, usrRi); err != nil {
		return err
	}
	log.Infow("Updated roster", "jid", ri.JID, "username", username, "xep", "roster")
	return nil
}

func (r *Roster) removeItem(ctx context.Context, ri *rostermodel.Item, userJID *jid.JID) error {
	var unsubscribe, unsubscribed *stravaganza.Presence

	contactJID, _ := jid.NewWithString(ri.JID, true)

	usrRi, err := r.rep.FetchRosterItem(ctx, userJID.Node(), ri.JID)
	if err != nil {
		return err
	}
	usrSub := rostermodel.None
	if usrRi != nil {
		usrSub = usrRi.Subscription
		switch usrSub {
		case rostermodel.To:
			unsubscribe = xmpputil.MakePresence(userJID, contactJID, stravaganza.UnsubscribeType, nil)
		case rostermodel.From:
			unsubscribed = xmpputil.MakePresence(userJID, contactJID, stravaganza.UnsubscribedType, nil)
		case rostermodel.Both:
			unsubscribe = xmpputil.MakePresence(userJID, contactJID, stravaganza.UnsubscribeType, nil)
			unsubscribed = xmpputil.MakePresence(userJID, contactJID, stravaganza.UnsubscribedType, nil)
		}
		usrRi.Subscription = rostermodel.Remove
		usrRi.Ask = false

		_, err := r.deleteNotification(ctx, contactJID.Node(), userJID)
		if err != nil {
			return err
		}
		if err := r.deleteItem(ctx, usrRi); err != nil {
			return err
		}
	}

	if r.hosts.IsLocalHost(contactJID.Domain()) {
		cntRi, err := r.rep.FetchRosterItem(ctx, contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			if cntRi.Subscription == rostermodel.From || cntRi.Subscription == rostermodel.Both {
				if err := r.routePresencesFrom(ctx, contactJID.Node(), userJID, stravaganza.UnavailableType); err != nil {
					return err
				}
			}
			switch cntRi.Subscription {
			case rostermodel.Both:
				cntRi.Subscription = rostermodel.To
				if err := r.upsertItem(ctx, cntRi); err != nil {
					return err
				}
				fallthrough

			default:
				cntRi.Subscription = rostermodel.None
				if err := r.upsertItem(ctx, cntRi); err != nil {
					return err
				}
			}
		}
	}
	if unsubscribe != nil {
		_, _ = r.router.Route(ctx, unsubscribe)
	}
	if unsubscribed != nil {
		_, _ = r.router.Route(ctx, unsubscribed)
	}

	if usrSub == rostermodel.From || usrSub == rostermodel.Both {
		if err := r.routePresencesFrom(ctx, userJID.Node(), contactJID, stravaganza.UnavailableType); err != nil {
			return err
		}
	}
	log.Infow("Removed roster item", "jid", contactJID, "username", userJID.Node(), "xep", "roster")
	return nil
}

func (r *Roster) upsertItem(ctx context.Context, ri *rostermodel.Item) error {
	err := r.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		ver, err := tx.TouchRosterVersion(ctx, ri.Username)
		if err != nil {
			return err
		}
		if err := tx.UpsertRosterItem(ctx, ri); err != nil {
			return err
		}
		return r.pushItem(ctx, ri, ver)
	})
	if err != nil {
		return err
	}
	return r.postRosterEvent(ctx, event.RosterItemUpdated, &event.RosterEventInfo{
		Username:     ri.Username,
		JID:          ri.JID,
		Subscription: ri.Subscription,
	})
}

func (r *Roster) deleteItem(ctx context.Context, ri *rostermodel.Item) error {
	err := r.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		ver, err := tx.TouchRosterVersion(ctx, ri.Username)
		if err != nil {
			return err
		}
		if err := tx.DeleteRosterItem(ctx, ri.Username, ri.JID); err != nil {
			return err
		}
		return r.pushItem(ctx, ri, ver)
	})
	if err != nil {
		return err
	}
	return r.postRosterEvent(ctx, event.RosterItemUpdated, &event.RosterEventInfo{
		Username:     ri.Username,
		JID:          ri.JID,
		Subscription: rostermodel.Remove,
	})
}

func (r *Roster) upsertNotification(ctx context.Context, contact string, userJID *jid.JID, presence *stravaganza.Presence) error {
	rn := &rostermodel.Notification{
		Contact:  contact,
		JID:      userJID.String(),
		Presence: presence,
	}
	return r.rep.UpsertRosterNotification(ctx, rn)
}

func (r *Roster) deleteNotification(ctx context.Context, contact string, userJID *jid.JID) (deleted bool, err error) {
	rn, err := r.rep.FetchRosterNotification(ctx, contact, userJID.String())
	if err != nil {
		return false, err
	}
	if rn == nil {
		return false, nil
	}
	if err := r.rep.DeleteRosterNotification(ctx, contact, userJID.String()); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Roster) pushItem(ctx context.Context, ri *rostermodel.Item, ver int) error {
	rss, err := r.resMng.GetResources(ctx, ri.Username)
	if err != nil {
		return err
	}
	for _, rs := range rss {
		// did request roster?
		if !rs.Info.Bool(requestedInfoKey) {
			continue
		}
		pushIQ, _ := stravaganza.NewIQBuilder().
			WithAttribute(stravaganza.ID, uuid.New().String()).
			WithAttribute(stravaganza.Type, stravaganza.SetType).
			WithAttribute(stravaganza.From, rs.JID.ToBareJID().String()).
			WithAttribute(stravaganza.To, rs.JID.String()).
			WithChild(
				stravaganza.NewBuilder("query").
					WithAttribute(stravaganza.Namespace, rosterNamespace).
					WithAttribute("ver", strconv.Itoa(ver)).
					WithChild(encodeRosterItem(ri)).
					Build(),
			).
			BuildIQ()

		_, _ = r.router.Route(ctx, pushIQ)
	}
	return nil
}

func (r *Roster) routePresencesFrom(ctx context.Context, username string, toJID *jid.JID, presenceType string) error {
	rss, err := r.resMng.GetResources(ctx, username)
	if err != nil {
		return err
	}
	for _, res := range rss {
		var children []stravaganza.Element
		if pr := res.Presence; pr != nil && pr.IsAvailable() {
			children = pr.AllChildren()
		}
		p := xmpputil.MakePresence(res.JID, toJID, presenceType, children)
		_, _ = r.router.Route(ctx, p)
	}
	return nil
}

func (r *Roster) setStreamValue(ctx context.Context, username, resource, key string, val interface{}) error {
	stm := r.router.C2S().LocalStream(username, resource)
	if stm == nil {
		return errStreamNotFound(username, resource)
	}
	return stm.SetInfoValue(ctx, key, val)
}

func (r *Roster) getStreamInfo(username, resource string) (inf c2smodel.Info, err error) {
	stm := r.router.C2S().LocalStream(username, resource)
	if stm == nil {
		return c2smodel.Info{}, errStreamNotFound(username, resource)
	}
	return stm.Info(), nil
}

func (r *Roster) postRosterEvent(ctx context.Context, eventName string, inf *event.RosterEventInfo) error {
	return r.sn.Post(ctx, sonar.NewEventBuilder(eventName).
		WithInfo(inf).
		WithSender(r).
		Build(),
	)
}

func decodeRosterItem(elem stravaganza.Element) (*rostermodel.Item, error) {
	if elem.Name() != "item" {
		return nil, fmt.Errorf("roster: invalid item element name: %s", elem.Name())
	}
	ri := &rostermodel.Item{}
	if jidStr := elem.Attribute("jid"); len(jidStr) > 0 {
		j, err := jid.NewWithString(jidStr, false)
		if err != nil {
			return nil, err
		}
		ri.JID = j.String()
	} else {
		return nil, errors.New("roster: item 'jid' attribute is required")
	}
	ri.Name = elem.Attribute("name")

	subscription := elem.Attribute("subscription")
	if len(subscription) > 0 {
		switch subscription {
		case rostermodel.Both, rostermodel.From, rostermodel.To, rostermodel.None, rostermodel.Remove:
			break
		default:
			return nil, fmt.Errorf("roster: unrecognized 'subscription' enum type: %s", subscription)
		}
		ri.Subscription = subscription
	}
	ask := elem.Attribute("ask")
	if len(ask) > 0 {
		if ask != "subscribe" {
			return nil, fmt.Errorf("roster: unrecognized 'ask' enum type: %s", subscription)
		}
		ri.Ask = true
	}
	groups := elem.Children("group")
	for _, group := range groups {
		if group.AttributeCount() > 0 {
			return nil, errors.New("roster: group element must not contain any attribute")
		}
		if len(group.Text()) > 0 {
			ri.Groups = append(ri.Groups, group.Text())
		}
	}
	return ri, nil
}

func encodeRosterItem(ri *rostermodel.Item) stravaganza.Element {
	b := stravaganza.NewBuilder("item").
		WithAttribute("name", ri.Name).
		WithAttribute("jid", ri.JID).
		WithAttribute("subscription", ri.Subscription)
	for _, group := range ri.Groups {
		b.WithChild(stravaganza.NewBuilder("group").
			WithText(group).
			Build(),
		)
	}
	return b.Build()
}

func parseVer(ver string) int {
	if len(ver) > 0 && ver[0] == 'v' {
		v, _ := strconv.Atoi(ver[1:])
		return v
	}
	return 0
}

func errStreamNotFound(username, resource string) error {
	return fmt.Errorf("roster: local stream not found: %s/%s", username, resource)
}
