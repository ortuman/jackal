/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
)

const mailboxSize = 2048

const rosterNamespace = "jabber:iq:roster"

const rosterRequestedCtxKey = "roster:requested"

// Config represents a roster configuration.
type Config struct {
	Versioning bool `yaml:"versioning"`
}

// Roster represents a roster server stream module.
type Roster struct {
	cfg        *Config
	router     *router.Router
	onlineJIDs sync.Map
	actorCh    chan func()
	shutdownCh chan chan bool
}

// New returns a roster server stream module.
func New(cfg *Config, router *router.Router) (*Roster, chan<- chan bool) {
	r := &Roster{
		cfg:        cfg,
		router:     router,
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan bool),
	}
	go r.loop()
	return r, r.shutdownCh
}

// MatchesIQ returns whether or not an IQ should be
// processed by the roster module.
func (r *Roster) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.Elements().ChildNamespace("query", rosterNamespace) != nil
}

// ProcessIQ processes a roster IQ taking according actions
// over the associated stream.
func (r *Roster) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	r.actorCh <- func() {
		if err := r.processIQ(iq, stm); err != nil {
			log.Error(err)
		}
	}
}

// ProcessPresence process an incoming roster presence.
func (r *Roster) ProcessPresence(presence *xmpp.Presence) {
	r.actorCh <- func() {
		if err := r.processPresence(presence); err != nil {
			log.Error(err)
		}
	}
}

// OnlinePresencesMatchingJID returns current online presences matching a given JID.
func (r *Roster) OnlinePresencesMatchingJID(j *jid.JID) []*xmpp.Presence {
	var ret []*xmpp.Presence
	r.onlineJIDs.Range(func(_, value interface{}) bool {
		switch presence := value.(type) {
		case *xmpp.Presence:
			if r.onlineJIDMatchesJID(presence.FromJID(), j) {
				ret = append(ret, presence)
			}
		}
		return true
	})
	return ret
}

// runs on it's own goroutine
func (r *Roster) loop() {
	for {
		select {
		case f := <-r.actorCh:
			f()
		case c := <-r.shutdownCh:
			c <- true
			return
		}
	}
}

func (r *Roster) processIQ(iq *xmpp.IQ, stm stream.C2S) error {
	var err error
	q := iq.Elements().ChildNamespace("query", rosterNamespace)
	if iq.IsGet() {
		err = r.sendRoster(iq, q, stm)
	} else if iq.IsSet() {
		err = r.updateRoster(iq, q, stm)
	} else {
		stm.SendElement(iq.BadRequestError())
	}
	return err
}

func (r *Roster) sendRoster(iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) error {
	if query.Elements().Count() > 0 {
		stm.SendElement(iq.BadRequestError())
		return nil
	}
	userJID := stm.JID()

	log.Infof("retrieving user roster... (%s)", userJID)

	itms, ver, err := storage.FetchRosterItems(userJID.Node())
	if err != nil {
		stm.SendElement(iq.InternalServerError())
		return err
	}
	v := r.parseVer(query.Attributes().Get("ver"))

	res := iq.ResultIQ()
	if v == 0 || v < ver.DeletionVer {
		// push all roster items
		q := xmpp.NewElementNamespace("query", rosterNamespace)
		if r.cfg.Versioning {
			q.SetAttribute("ver", fmt.Sprintf("v%d", ver.Ver))
		}
		for _, itm := range itms {
			q.AppendElement(itm.Element())
		}
		res.AppendElement(q)
		stm.SendElement(res)
	} else {
		// push roster changes
		stm.SendElement(res)
		for _, itm := range itms {
			if itm.Ver > v {
				iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
				q := xmpp.NewElementNamespace("query", rosterNamespace)
				q.SetAttribute("ver", fmt.Sprintf("v%d", itm.Ver))
				q.AppendElement(itm.Element())
				iq.AppendElement(q)
				stm.SendElement(iq)
			}
		}
	}
	stm.SetBool(rosterRequestedCtxKey, true)
	return nil
}

func (r *Roster) updateRoster(iq *xmpp.IQ, query xmpp.XElement, stm stream.C2S) error {
	itms := query.Elements().Children("item")
	if len(itms) != 1 {
		stm.SendElement(iq.BadRequestError())
		return nil
	}
	ri, err := rostermodel.NewItem(itms[0])
	if err != nil {
		stm.SendElement(iq.BadRequestError())
		return err
	}
	switch ri.Subscription {
	case rostermodel.SubscriptionRemove:
		if err := r.removeItem(ri, stm); err != nil {
			stm.SendElement(iq.InternalServerError())
			return err
		}
	default:
		if err := r.updateItem(ri, stm); err != nil {
			stm.SendElement(iq.InternalServerError())
			return err
		}
	}
	stm.SendElement(iq.ResultIQ())
	return nil
}

func (r *Roster) updateItem(ri *rostermodel.Item, stm stream.C2S) error {
	userJID := stm.JID().ToBareJID()
	contactJID := ri.ContactJID()

	log.Infof("updating roster item - contact: %s (%s)", contactJID, userJID)

	usrRi, err := storage.FetchRosterItem(userJID.Node(), contactJID.String())
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
			Username:     userJID.Node(),
			JID:          ri.JID,
			Name:         ri.Name,
			Subscription: rostermodel.SubscriptionNone,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	return r.insertItem(usrRi, userJID)
}

func (r *Roster) removeItem(ri *rostermodel.Item, stm stream.C2S) error {
	var unsubscribe, unsubscribed *xmpp.Presence

	userJID := stm.JID().ToBareJID()
	contactJID := ri.ContactJID()

	log.Infof("removing roster item: %v (%s)", contactJID, userJID)

	usrRi, err := storage.FetchRosterItem(userJID.Node(), contactJID.String())
	if err != nil {
		return err
	}
	usrSub := rostermodel.SubscriptionNone
	if usrRi != nil {
		usrSub = usrRi.Subscription
		switch usrSub {
		case rostermodel.SubscriptionTo:
			unsubscribe = xmpp.NewPresence(userJID, contactJID, xmpp.UnsubscribeType)
		case rostermodel.SubscriptionFrom:
			unsubscribed = xmpp.NewPresence(userJID, contactJID, xmpp.UnsubscribedType)
		case rostermodel.SubscriptionBoth:
			unsubscribe = xmpp.NewPresence(userJID, contactJID, xmpp.UnsubscribeType)
			unsubscribed = xmpp.NewPresence(userJID, contactJID, xmpp.UnsubscribedType)
		}
		usrRi.Subscription = rostermodel.SubscriptionRemove
		usrRi.Ask = false

		_, err := r.deleteNotification(contactJID.Node(), userJID)
		if err != nil {
			return err
		}
		if err := r.deleteItem(usrRi, userJID); err != nil {
			return err
		}
	}
	if r.router.IsLocalHost(contactJID.Domain()) {
		cntRi, err := storage.FetchRosterItem(contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			if cntRi.Subscription == rostermodel.SubscriptionFrom || cntRi.Subscription == rostermodel.SubscriptionBoth {
				r.routePresencesFrom(contactJID, userJID, xmpp.UnavailableType)
			}
			switch cntRi.Subscription {
			case rostermodel.SubscriptionBoth:
				cntRi.Subscription = rostermodel.SubscriptionTo
				if r.insertItem(cntRi, contactJID); err != nil {
					return err
				}
				fallthrough

			default:
				cntRi.Subscription = rostermodel.SubscriptionNone
				if r.insertItem(cntRi, contactJID); err != nil {
					return err
				}
			}
		}
	}
	if unsubscribe != nil {
		r.router.Route(unsubscribe)
	}
	if unsubscribed != nil {
		r.router.Route(unsubscribed)
	}

	if usrSub == rostermodel.SubscriptionFrom || usrSub == rostermodel.SubscriptionBoth {
		r.routePresencesFrom(userJID, contactJID, xmpp.UnavailableType)
	}
	return nil
}

func (r *Roster) processPresence(presence *xmpp.Presence) error {
	switch presence.Type() {
	case xmpp.SubscribeType:
		return r.processSubscribe(presence)
	case xmpp.SubscribedType:
		return r.processSubscribed(presence)
	case xmpp.UnsubscribeType:
		return r.processUnsubscribe(presence)
	case xmpp.UnsubscribedType:
		return r.processUnsubscribed(presence)
	case xmpp.ProbeType:
		return r.processProbePresence(presence)
	case xmpp.AvailableType, xmpp.UnavailableType:
		return r.processAvailablePresence(presence)
	}
	return nil
}

func (r *Roster) processSubscribe(presence *xmpp.Presence) error {
	userJID := presence.FromJID().ToBareJID()
	contactJID := presence.ToJID().ToBareJID()

	log.Infof("processing 'subscribe' - contact: %s (%s)", contactJID, userJID)

	if r.router.IsLocalHost(userJID.Domain()) {
		usrRi, err := storage.FetchRosterItem(userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case rostermodel.SubscriptionTo, rostermodel.SubscriptionBoth:
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
				Subscription: rostermodel.SubscriptionNone,
				Ask:          true,
			}
		}
		if r.insertItem(usrRi, userJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "subscribe" with the user's bare JID as the 'from' address
	p := xmpp.NewPresence(userJID, contactJID, xmpp.SubscribeType)
	p.AppendElements(presence.Elements().All())

	if r.router.IsLocalHost(contactJID.Domain()) {
		// archive roster approval notification
		if err := r.insertOrUpdateNotification(contactJID.Node(), userJID, p); err != nil {
			return err
		}
	}
	r.router.Route(p)
	return nil
}

func (r *Roster) processSubscribed(presence *xmpp.Presence) error {
	userJID := presence.ToJID().ToBareJID()
	contactJID := presence.FromJID().ToBareJID()

	log.Infof("processing 'subscribed' - user: %s (%s)", userJID, contactJID)

	if r.router.IsLocalHost(contactJID.Domain()) {
		_, err := r.deleteNotification(contactJID.Node(), userJID)
		if err != nil {
			return err
		}
		cntRi, err := storage.FetchRosterItem(contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			switch cntRi.Subscription {
			case rostermodel.SubscriptionTo:
				cntRi.Subscription = rostermodel.SubscriptionBoth
			case rostermodel.SubscriptionNone:
				cntRi.Subscription = rostermodel.SubscriptionFrom
			}
		} else {
			// create roster item if not previously created
			cntRi = &rostermodel.Item{
				Username:     contactJID.Node(),
				JID:          userJID.String(),
				Subscription: rostermodel.SubscriptionFrom,
				Ask:          false,
			}
		}
		if r.insertItem(cntRi, contactJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "subscribed" with the contact's bare JID as the 'from' address
	p := xmpp.NewPresence(contactJID, userJID, xmpp.SubscribedType)
	p.AppendElements(presence.Elements().All())

	if r.router.IsLocalHost(userJID.Domain()) {
		usrRi, err := storage.FetchRosterItem(userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case rostermodel.SubscriptionFrom:
				usrRi.Subscription = rostermodel.SubscriptionBoth
			case rostermodel.SubscriptionNone:
				usrRi.Subscription = rostermodel.SubscriptionTo
			default:
				return nil
			}
			usrRi.Ask = false
			if r.insertItem(usrRi, userJID); err != nil {
				return err
			}
		}
	}
	r.router.Route(p)
	r.routePresencesFrom(contactJID, userJID, xmpp.AvailableType)
	return nil
}

func (r *Roster) processUnsubscribe(presence *xmpp.Presence) error {
	userJID := presence.FromJID().ToBareJID()
	contactJID := presence.ToJID().ToBareJID()

	log.Infof("processing 'unsubscribe' - contact: %s (%s)", contactJID, userJID)

	var usrSub string
	if r.router.IsLocalHost(userJID.Domain()) {
		usrRi, err := storage.FetchRosterItem(userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		usrSub = rostermodel.SubscriptionNone
		if usrRi != nil {
			usrSub = usrRi.Subscription
			switch usrSub {
			case rostermodel.SubscriptionBoth:
				usrRi.Subscription = rostermodel.SubscriptionFrom
			default:
				usrRi.Subscription = rostermodel.SubscriptionNone
			}
			if r.insertItem(usrRi, userJID); err != nil {
				return err
			}
		}
	}
	// stamp the presence stanza of type "unsubscribe" with the users's bare JID as the 'from' address
	p := xmpp.NewPresence(userJID, contactJID, xmpp.UnsubscribeType)
	p.AppendElements(presence.Elements().All())

	if r.router.IsLocalHost(contactJID.Domain()) {
		cntRi, err := storage.FetchRosterItem(contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			switch cntRi.Subscription {
			case rostermodel.SubscriptionBoth:
				cntRi.Subscription = rostermodel.SubscriptionTo
			default:
				cntRi.Subscription = rostermodel.SubscriptionNone
			}
			if r.insertItem(cntRi, contactJID); err != nil {
				return err
			}
		}
	}
	r.router.Route(p)

	if usrSub == rostermodel.SubscriptionTo || usrSub == rostermodel.SubscriptionBoth {
		r.routePresencesFrom(contactJID, userJID, xmpp.UnavailableType)
	}
	return nil
}

func (r *Roster) processUnsubscribed(presence *xmpp.Presence) error {
	userJID := presence.ToJID().ToBareJID()
	contactJID := presence.FromJID().ToBareJID()

	log.Infof("processing 'unsubscribed' - user: %s (%s)", userJID, contactJID)

	var cntSub string
	if r.router.IsLocalHost(contactJID.Domain()) {
		deleted, err := r.deleteNotification(contactJID.Node(), userJID)
		if err != nil {
			return err
		}
		// do not change subscription state if cancelling a subscription request
		if deleted {
			goto routePresence
		}
		cntRi, err := storage.FetchRosterItem(contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		cntSub = rostermodel.SubscriptionNone
		if cntRi != nil {
			cntSub = cntRi.Subscription
			switch cntSub {
			case rostermodel.SubscriptionBoth:
				cntRi.Subscription = rostermodel.SubscriptionTo
			default:
				cntRi.Subscription = rostermodel.SubscriptionNone
			}
			if r.insertItem(cntRi, contactJID); err != nil {
				return err
			}
		}
	}
routePresence:
	// stamp the presence stanza of type "unsubscribed" with the contact's bare JID as the 'from' address
	p := xmpp.NewPresence(contactJID, userJID, xmpp.UnsubscribedType)
	p.AppendElements(presence.Elements().All())

	if r.router.IsLocalHost(userJID.Domain()) {
		usrRi, err := storage.FetchRosterItem(userJID.Node(), contactJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			if !usrRi.Ask { // pending out...
				switch usrRi.Subscription {
				case rostermodel.SubscriptionBoth:
					usrRi.Subscription = rostermodel.SubscriptionFrom
				default:
					usrRi.Subscription = rostermodel.SubscriptionNone
				}
			}
			usrRi.Ask = false
			if r.insertItem(usrRi, userJID); err != nil {
				return err
			}
		}
	}
	r.router.Route(p)

	if cntSub == rostermodel.SubscriptionFrom || cntSub == rostermodel.SubscriptionBoth {
		r.routePresencesFrom(contactJID, userJID, xmpp.UnavailableType)
	}
	return nil
}

func (r *Roster) processProbePresence(presence *xmpp.Presence) error {
	userJID := presence.ToJID().ToBareJID()
	contactJID := presence.FromJID().ToBareJID()

	log.Infof("processing 'probe' - user: %s (%s)", userJID, contactJID)

	ri, err := storage.FetchRosterItem(userJID.Node(), contactJID.String())
	if err != nil {
		return err
	}
	usr, err := storage.FetchUser(userJID.Node())
	if err != nil {
		return err
	}
	if usr == nil || ri == nil || (ri.Subscription != rostermodel.SubscriptionBoth && ri.Subscription != rostermodel.SubscriptionFrom) {
		r.router.Route(xmpp.NewPresence(userJID, contactJID, xmpp.UnsubscribedType))
		return nil
	}
	if usr.LastPresence != nil {
		p := xmpp.NewPresence(usr.LastPresence.FromJID(), contactJID, usr.LastPresence.Type())
		p.AppendElements(usr.LastPresence.Elements().All())
		r.router.Route(p)
	}
	return nil
}

func (r *Roster) processAvailablePresence(presence *xmpp.Presence) error {
	fromJID := presence.FromJID()

	userJID := fromJID.ToBareJID()
	contactJID := presence.ToJID().ToBareJID()

	replyOnBehalf := r.router.IsLocalHost(userJID.Domain()) && userJID.Matches(contactJID, jid.MatchesBare)

	// keep track of available presences
	if presence.IsAvailable() {
		log.Infof("processing 'available' - user: %s", fromJID)
		if _, loaded := r.onlineJIDs.LoadOrStore(fromJID.String(), presence); !loaded {
			if replyOnBehalf {
				if err := r.deliverRosterPresences(userJID); err != nil {
					return err
				}
			}
		}
	} else {
		log.Infof("processing 'unavailable' - user: %s", fromJID)
		r.onlineJIDs.Delete(fromJID.String())
	}
	if replyOnBehalf {
		return r.broadcastPresence(presence)
	}
	return r.router.Route(presence)
}

func (r *Roster) deliverRosterPresences(userJID *jid.JID) error {
	// first, deliver pending approval notifications...
	rns, err := storage.FetchRosterNotifications(userJID.Node())
	if err != nil {
		return err
	}
	for _, rn := range rns {
		fromJID, _ := jid.NewWithString(rn.JID, true)
		p := xmpp.NewPresence(fromJID, userJID, xmpp.SubscribeType)
		p.AppendElements(rn.Presence.Elements().All())
		r.router.Route(p)
	}

	// deliver roster online presences
	items, _, err := storage.FetchRosterItems(userJID.Node())
	if err != nil {
		return err
	}
	for _, item := range items {
		switch item.Subscription {
		case rostermodel.SubscriptionTo, rostermodel.SubscriptionBoth:
			contactJID := item.ContactJID()
			if !r.router.IsLocalHost(contactJID.Domain()) {
				r.router.Route(xmpp.NewPresence(userJID, contactJID, xmpp.ProbeType))
				continue
			}
			r.routePresencesFrom(contactJID, userJID, xmpp.AvailableType)
		}
	}
	return nil
}

func (r *Roster) broadcastPresence(presence *xmpp.Presence) error {
	fromJID := presence.FromJID()
	itms, _, err := storage.FetchRosterItems(fromJID.Node())
	if err != nil {
		return err
	}
	for _, itm := range itms {
		switch itm.Subscription {
		case rostermodel.SubscriptionFrom, rostermodel.SubscriptionBoth:
			p := xmpp.NewPresence(fromJID, itm.ContactJID(), presence.Type())
			p.AppendElements(presence.Elements().All())
			r.router.Route(p)
		}
	}

	// update last received presence
	if usr, err := storage.FetchUser(fromJID.Node()); err != nil {
		return err
	} else if usr != nil {
		usr.LastPresence = presence
		return storage.InsertOrUpdateUser(usr)
	}
	return nil
}

func (r *Roster) onlineJIDMatchesJID(onlineJID, j *jid.JID) bool {
	if j.IsFullWithUser() {
		return onlineJID.Matches(j, jid.MatchesNode|jid.MatchesDomain|jid.MatchesResource)
	} else if j.IsFullWithServer() {
		return onlineJID.Matches(j, jid.MatchesDomain|jid.MatchesResource)
	} else if j.IsBare() {
		return onlineJID.Matches(j, jid.MatchesNode|jid.MatchesDomain)
	}
	return onlineJID.Matches(j, jid.MatchesDomain)
}

func (r *Roster) insertItem(ri *rostermodel.Item, pushTo *jid.JID) error {
	v, err := storage.InsertOrUpdateRosterItem(ri)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return r.pushItem(ri, pushTo)
}

func (r *Roster) deleteItem(ri *rostermodel.Item, pushTo *jid.JID) error {
	v, err := storage.DeleteRosterItem(ri.Username, ri.JID)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return r.pushItem(ri, pushTo)
}

func (r *Roster) pushItem(ri *rostermodel.Item, to *jid.JID) error {
	query := xmpp.NewElementNamespace("query", rosterNamespace)
	if r.cfg.Versioning {
		query.SetAttribute("ver", fmt.Sprintf("v%d", ri.Ver))
	}
	query.AppendElement(ri.Element())

	stms := r.router.UserStreams(to.Node())
	for _, stm := range stms {
		if !stm.GetBool(rosterRequestedCtxKey) {
			continue
		}
		pushEl := xmpp.NewIQType(uuid.New(), xmpp.SetType)
		pushEl.SetTo(stm.JID().String())
		pushEl.AppendElement(query)
		stm.SendElement(pushEl)
	}
	return nil
}

func (r *Roster) deleteNotification(contact string, userJID *jid.JID) (deleted bool, err error) {
	rn, err := storage.FetchRosterNotification(contact, userJID.String())
	if err != nil {
		return false, err
	}
	if rn == nil {
		return false, nil
	}
	if err := storage.DeleteRosterNotification(contact, userJID.String()); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Roster) insertOrUpdateNotification(contact string, userJID *jid.JID, presence *xmpp.Presence) error {
	rn := &rostermodel.Notification{
		Contact:  contact,
		JID:      userJID.String(),
		Presence: presence,
	}
	return storage.InsertOrUpdateRosterNotification(rn)
}

func (r *Roster) routePresencesFrom(from *jid.JID, to *jid.JID, presenceType string) {
	stms := r.router.UserStreams(from.Node())
	for _, stm := range stms {
		p := xmpp.NewPresence(stm.JID(), to.ToBareJID(), presenceType)
		if presence := stm.Presence(); presence != nil && presence.IsAvailable() {
			p.AppendElements(presence.Elements().All())
		}
		r.router.Route(p)
	}
}

func (r *Roster) parseVer(ver string) int {
	if len(ver) > 0 && ver[0] == 'v' {
		v, _ := strconv.Atoi(ver[1:])
		return v
	}
	return 0
}
