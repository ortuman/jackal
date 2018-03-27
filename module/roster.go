/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const rosterNamespace = "jabber:iq:roster"

const (
	subscriptionNone   = "none"
	subscriptionFrom   = "from"
	subscriptionTo     = "to"
	subscriptionBoth   = "both"
	subscriptionRemove = "remove"
)

type roster struct {
	mu    sync.RWMutex
	items []model.RosterItem
}

func (r *roster) fetchItems() []model.RosterItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.items
}

func (r *roster) fetchItem(contact string) *model.RosterItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := 0; i < len(r.items); i++ {
		if r.items[i].Contact == contact {
			return &r.items[i]
		}
	}
	return nil
}

func (r *roster) insertOrUpdateItem(ri *model.RosterItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var found bool
	for i := 0; !found && i < len(r.items); i++ {
		if r.items[i].Contact == ri.Contact {
			r.items[i] = *ri
			found = true
		}
	}
	if !found {
		r.items = append(r.items, *ri)
	}
}

func (r *roster) deleteItem(ri *model.RosterItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var newItems []model.RosterItem
	for i := 0; i < len(r.items); i++ {
		if r.items[i].Contact != ri.Contact {
			newItems = append(newItems, r.items[i])
		}
	}
	r.items = newItems
}

type rosterMap struct {
	mu    sync.RWMutex
	cache map[string]*roster
}

func (rm *rosterMap) loadRoster(username string) error {
	rm.mu.RLock()
	r := rm.cache[username]
	rm.mu.RUnlock()
	if r != nil {
		return nil
	}
	items, err := storage.Instance().FetchRosterItems(username)
	if err != nil {
		return err
	}
	rm.mu.Lock()
	rm.cache[username] = &roster{items: items}
	rm.mu.Unlock()
	return nil
}

func (rm *rosterMap) fetchRosterItems(username string) ([]model.RosterItem, error) {
	rm.mu.RLock()
	r := rm.cache[username]
	rm.mu.RUnlock()
	if r != nil {
		return r.fetchItems(), nil
	}
	return storage.Instance().FetchRosterItems(username)
}

func (rm *rosterMap) fetchRosterItem(username, contact string) (*model.RosterItem, error) {
	rm.mu.RLock()
	r := rm.cache[username]
	rm.mu.RUnlock()
	if r != nil {
		return r.fetchItem(contact), nil
	}
	return storage.Instance().FetchRosterItem(username, contact)
}

func (rm *rosterMap) insertOrUpdateRosterItem(ri *model.RosterItem) error {
	rm.mu.RLock()
	r := rm.cache[ri.User]
	rm.mu.RUnlock()
	if r != nil {
		r.insertOrUpdateItem(ri)
	}
	return storage.Instance().InsertOrUpdateRosterItem(ri)
}

func (rm *rosterMap) deleteRosterItem(ri *model.RosterItem) error {
	rm.mu.RLock()
	r := rm.cache[ri.User]
	rm.mu.RUnlock()
	if r != nil {
		r.deleteItem(ri)
	}
	return storage.Instance().DeleteRosterItem(ri.User, ri.Contact)
}

func (rm *rosterMap) unloadRoster(username string) {
	rm.mu.Lock()
	delete(rm.cache, username)
	rm.mu.Unlock()
}

var rosterTable = rosterMap{
	cache: map[string]*roster{},
}

var defaultRosterErrHandler = func(err error) {
	log.Error(err)
}

// ModRoster represents a roster server stream module.
type ModRoster struct {
	stm        c2s.Stream
	lock       sync.RWMutex
	requested  bool
	actorCh    chan func()
	doneCh     chan chan bool
	errHandler func(error)
}

// NewRoster returns a roster server stream module.
func NewRoster(stm c2s.Stream) *ModRoster {
	r := &ModRoster{
		stm:        stm,
		actorCh:    make(chan func(), moduleMailboxSize),
		doneCh:     make(chan chan bool),
		errHandler: defaultRosterErrHandler,
	}
	go r.actorLoop()
	return r
}

// AssociatedNamespaces returns namespaces associated
// with roster module.
func (r *ModRoster) AssociatedNamespaces() []string {
	return []string{}
}

// Done signals stream termination.
func (r *ModRoster) Done() {
	ch := make(chan bool)
	r.doneCh <- ch
	<-ch // wait until closed...
}

// MatchesIQ returns whether or not an IQ should be
// processed by the roster module.
func (r *ModRoster) MatchesIQ(iq *xml.IQ) bool {
	return iq.Elements().ChildNamespace("query", rosterNamespace) != nil
}

// ProcessIQ processes a roster IQ taking according actions
// over the associated stream.
func (r *ModRoster) ProcessIQ(iq *xml.IQ) {
	r.actorCh <- func() {
		q := iq.Elements().ChildNamespace("query", rosterNamespace)
		if iq.IsGet() {
			r.sendRoster(iq, q)
		} else if iq.IsSet() {
			r.updateRoster(iq, q)
		} else {
			r.stm.SendElement(iq.BadRequestError())
		}
	}
}

// IsRequested returns whether or not the user roster
// has been requested.
func (r *ModRoster) IsRequested() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.requested
}

// ProcessPresence process an incoming roster presence.
func (r *ModRoster) ProcessPresence(presence *xml.Presence) {
	r.actorCh <- func() {
		if err := r.processPresence(presence); err != nil {
			r.errHandler(err)
		}
	}
}

// DeliverPendingApprovalNotifications delivers any pending roster notification
// to the associated stream.
func (r *ModRoster) DeliverPendingApprovalNotifications() {
	r.actorCh <- func() {
		if err := r.deliverPendingApprovalNotifications(); err != nil {
			r.errHandler(err)
		}
	}
}

// ReceivePresences delivers all inbound roster available presences
// to the associated module stream.
func (r *ModRoster) ReceivePresences() {
	r.actorCh <- func() {
		if err := r.receivePresences(); err != nil {
			r.errHandler(err)
		}
	}
}

// BroadcastPresence broadcasts presence to all outbound roster contacts.
func (r *ModRoster) BroadcastPresence(presence *xml.Presence) {
	r.actorCh <- func() {
		if err := r.broadcastPresence(presence); err != nil {
			r.errHandler(err)
		}
	}
}

// BroadcastPresenceAndWait broadcasts presence to all outbound
// roster contacts in a synchronous manner.
func (r *ModRoster) BroadcastPresenceAndWait(presence *xml.Presence) {
	continueCh := make(chan struct{})
	r.actorCh <- func() {
		if err := r.broadcastPresence(presence); err != nil {
			r.errHandler(err)
		}
		close(continueCh)
	}
	<-continueCh
}

func (r *ModRoster) actorLoop() {
	for {
		select {
		case f := <-r.actorCh:
			f()
		case ch := <-r.doneCh:
			defer close(ch)
			rosterTable.unloadRoster(r.stm.Username())
			return
		}
	}
}

func (r *ModRoster) processPresence(presence *xml.Presence) error {
	switch presence.Type() {
	case xml.SubscribeType:
		return r.processSubscribe(presence)
	case xml.SubscribedType:
		return r.processSubscribed(presence)
	case xml.UnsubscribeType:
		return r.processUnsubscribe(presence)
	case xml.UnsubscribedType:
		return r.processUnsubscribed(presence)
	}
	return nil
}

func (r *ModRoster) deliverPendingApprovalNotifications() error {
	rosterNotifications, err := storage.Instance().FetchRosterNotifications(r.stm.Username())
	if err != nil {
		return err
	}
	for _, rosterNotification := range rosterNotifications {
		fromJID, _ := xml.NewJID(rosterNotification.User, r.stm.Domain(), "", true)
		p := xml.NewPresence(fromJID, r.stm.JID(), xml.SubscribeType)
		p.AppendElements(rosterNotification.Elements)
		r.stm.SendElement(p)
	}
	return nil
}

func (r *ModRoster) receivePresences() error {
	items, err := rosterTable.fetchRosterItems(r.stm.JID().Node())
	if err != nil {
		return err
	}
	userJID := r.stm.JID()
	for _, item := range items {
		switch item.Subscription {
		case subscriptionTo, subscriptionBoth:
			r.routePresencesFrom(r.rosterItemJID(&item), userJID, xml.AvailableType)
		}
	}
	return nil
}

func (r *ModRoster) broadcastPresence(presence *xml.Presence) error {
	items, err := rosterTable.fetchRosterItems(r.stm.JID().Node())
	if err != nil {
		return err
	}
	for _, item := range items {
		switch item.Subscription {
		case subscriptionFrom, subscriptionBoth:
			r.routePresence(presence, r.rosterItemJID(&item))
		}
	}
	return nil
}

func (r *ModRoster) sendRoster(iq *xml.IQ, query xml.Element) {
	if query.Elements().Count() > 0 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("retrieving user roster... (%s/%s)", r.stm.Username(), r.stm.Resource())

	result := iq.ResultIQ()
	q := xml.NewElementNamespace("query", rosterNamespace)

	if err := rosterTable.loadRoster(r.stm.Username()); err != nil {
		r.errHandler(err)
		r.stm.SendElement(iq.InternalServerError())
		return
	}
	items, err := rosterTable.fetchRosterItems(r.stm.JID().Node())
	if err != nil {
		r.errHandler(err)
		r.stm.SendElement(iq.InternalServerError())
		return
	}
	for _, item := range items {
		q.AppendElement(r.elementFromRosterItem(&item))
	}
	result.AppendElement(q)
	r.stm.SendElement(result)

	r.lock.Lock()
	r.requested = true
	r.lock.Unlock()
}

func (r *ModRoster) updateRoster(iq *xml.IQ, query xml.Element) {
	items := query.Elements().Children("item")
	if len(items) != 1 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	ri, err := r.rosterItemFromElement(items[0])
	if err != nil {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	switch ri.Subscription {
	case subscriptionRemove:
		if err := r.removeRosterItem(ri); err != nil {
			r.errHandler(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	default:
		if err := r.updateRosterItem(ri); err != nil {
			r.errHandler(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	}
	r.stm.SendElement(iq.ResultIQ())
}

func (r *ModRoster) removeRosterItem(ri *model.RosterItem) error {
	userJID := r.stm.JID()
	contactJID := r.rosterItemJID(ri)

	log.Infof("removing roster item: %v (%s/%s)", contactJID, r.stm.Username(), r.stm.Resource())

	var unsubscribe *xml.Presence
	var unsubscribed *xml.Presence

	userRi, err := rosterTable.fetchRosterItem(userJID.Node(), contactJID.Node())
	if err != nil {
		return err
	}
	userSubscription := subscriptionNone
	if userRi != nil {
		userSubscription = userRi.Subscription
		switch userSubscription {
		case subscriptionTo:
			unsubscribe = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
		case subscriptionFrom:
			unsubscribed = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribedType)
		case subscriptionBoth:
			unsubscribe = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
			unsubscribed = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribedType)
		}
		userRi.Subscription = subscriptionRemove
		userRi.Ask = false

		if err := r.deleteRosterNotification(userJID, contactJID); err != nil {
			return err
		}
		if err := rosterTable.deleteRosterItem(userRi); err != nil {
			return err
		}
		if err := r.pushRosterItem(userRi, userJID); err != nil {
			return err
		}
	}

	if r.isLocalJID(contactJID) {
		contactRi, err := rosterTable.fetchRosterItem(contactJID.Node(), userJID.Node())
		if err != nil {
			return err
		}
		if contactRi != nil {
			if contactRi.Subscription == subscriptionFrom || contactRi.Subscription == subscriptionBoth {
				r.routePresencesFrom(contactJID, userJID, xml.UnavailableType)
			}
			switch contactRi.Subscription {
			case subscriptionBoth:
				contactRi.Subscription = subscriptionTo
				if err := r.pushRosterItem(contactRi, contactJID); err != nil {
					return err
				}
				fallthrough
			default:
				contactRi.Subscription = subscriptionNone
				if err := r.pushRosterItem(contactRi, contactJID); err != nil {
					return err
				}
			}
			if err := rosterTable.insertOrUpdateRosterItem(contactRi); err != nil {
				return err
			}
		}
	}
	if unsubscribe != nil {
		r.routePresence(unsubscribe, contactJID)
	}
	if unsubscribed != nil {
		r.routePresence(unsubscribed, contactJID)
	}
	if userSubscription == subscriptionFrom || userSubscription == subscriptionBoth {
		r.routePresencesFrom(userJID, contactJID, xml.UnavailableType)
	}
	return nil
}

func (r *ModRoster) updateRosterItem(ri *model.RosterItem) error {
	userJID := r.stm.JID()
	contactJID := r.rosterItemJID(ri)

	log.Infof("updating roster item - contact: %s (%s/%s)", contactJID, r.stm.Username(), r.stm.Resource())

	userRi, err := rosterTable.fetchRosterItem(userJID.Node(), contactJID.Node())
	if err != nil {
		return err
	}
	if userRi != nil {
		// update roster item
		if len(ri.Name) > 0 {
			userRi.Name = ri.Name
		}
		userRi.Groups = ri.Groups

	} else {
		userRi = &model.RosterItem{
			User:         r.stm.Username(),
			Contact:      ri.Contact,
			Name:         ri.Name,
			Subscription: subscriptionNone,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	if err := rosterTable.insertOrUpdateRosterItem(userRi); err != nil {
		return err
	}
	return r.pushRosterItem(userRi, r.stm.JID())
}

func (r *ModRoster) processSubscribe(presence *xml.Presence) error {
	userJID := r.stm.JID()
	contactJID := presence.ToJID()

	log.Infof("processing 'subscribe' - contact: %s (%s/%s)", contactJID, r.stm.Username(), r.stm.Resource())

	userRi, err := rosterTable.fetchRosterItem(userJID.Node(), contactJID.Node())
	if err != nil {
		return err
	}
	if userRi != nil {
		switch userRi.Subscription {
		case subscriptionTo, subscriptionBoth:
			return nil // already subscribed...
		default:
			if !userRi.Ask {
				userRi.Ask = true
			} else {
				return nil // notification already sent...
			}
		}
	} else {
		// create roster item if not previously created
		userRi = &model.RosterItem{
			User:         userJID.Node(),
			Contact:      contactJID.Node(),
			Subscription: subscriptionNone,
			Ask:          true,
		}
	}
	if err := rosterTable.insertOrUpdateRosterItem(userRi); err != nil {
		return err
	}
	if err := r.pushRosterItem(userRi, userJID); err != nil {
		return err
	}
	// stamp the presence stanza of type "subscribe" with the user's bare JID as the 'from' address
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.SubscribeType)
	p.AppendElements(presence.Elements().All())

	if r.isLocalJID(contactJID) {
		// archive roster approval notification
		if err := r.insertOrUpdateRosterNotification(userJID, contactJID, p); err != nil {
			return err
		}
	}
	r.routePresence(p, contactJID)
	return nil
}

func (r *ModRoster) processSubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.stm.JID()

	log.Infof("processing 'subscribed' - user: %s (%s/%s)", userJID, r.stm.Username(), r.stm.Resource())

	if err := r.deleteRosterNotification(userJID, contactJID); err != nil {
		return err
	}
	contactRi, err := rosterTable.fetchRosterItem(contactJID.Node(), userJID.Node())
	if err != nil {
		return err
	}
	if contactRi != nil {
		switch contactRi.Subscription {
		case subscriptionTo:
			contactRi.Subscription = subscriptionBoth
		case subscriptionNone:
			contactRi.Subscription = subscriptionFrom
		}
	} else {
		// create roster item if not previously created
		contactRi = &model.RosterItem{
			User:         contactJID.Node(),
			Contact:      userJID.Node(),
			Subscription: subscriptionFrom,
			Ask:          false,
		}
	}
	if err := rosterTable.insertOrUpdateRosterItem(contactRi); err != nil {
		return err
	}
	if err := r.pushRosterItem(contactRi, contactJID); err != nil {
		return err
	}
	// stamp the presence stanza of type "subscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.SubscribedType)
	p.AppendElements(presence.Elements().All())

	if r.isLocalJID(userJID) {
		userRi, err := rosterTable.fetchRosterItem(userJID.Node(), contactJID.Node())
		if err != nil {
			return err
		}
		if userRi != nil {
			switch userRi.Subscription {
			case subscriptionFrom:
				userRi.Subscription = subscriptionBoth
			case subscriptionNone:
				userRi.Subscription = subscriptionTo
			default:
				return nil
			}
			userRi.Ask = false
			if err := rosterTable.insertOrUpdateRosterItem(userRi); err != nil {
				return err
			}
			if err := r.pushRosterItem(userRi, userJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, userJID)
	r.routePresencesFrom(contactJID, userJID, xml.AvailableType)
	return nil
}

func (r *ModRoster) processUnsubscribe(presence *xml.Presence) error {
	userJID := r.stm.JID()
	contactJID := presence.ToJID()

	log.Infof("processing 'unsubscribe' - contact: %s (%s/%s)", contactJID, r.stm.Username(), r.stm.Resource())

	userRi, err := rosterTable.fetchRosterItem(userJID.Node(), contactJID.Node())
	if err != nil {
		return err
	}
	userSubscription := subscriptionNone
	if userRi != nil {
		userSubscription = userRi.Subscription
		switch userSubscription {
		case subscriptionBoth:
			userRi.Subscription = subscriptionFrom
		default:
			userRi.Subscription = subscriptionNone
		}
		if err := rosterTable.insertOrUpdateRosterItem(userRi); err != nil {
			return err
		}
		if err := r.pushRosterItem(userRi, userJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribe" with the users's bare JID as the 'from' address
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
	p.AppendElements(presence.Elements().All())

	if r.isLocalJID(contactJID) {
		contactRi, err := rosterTable.fetchRosterItem(contactJID.Node(), userJID.Node())
		if err != nil {
			return err
		}
		if contactRi != nil {
			switch contactRi.Subscription {
			case subscriptionBoth:
				contactRi.Subscription = subscriptionTo
			default:
				contactRi.Subscription = subscriptionNone
			}
			if err := rosterTable.insertOrUpdateRosterItem(contactRi); err != nil {
				return err
			}
			if err := r.pushRosterItem(contactRi, contactJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, contactJID)

	if userSubscription == subscriptionTo || userSubscription == subscriptionBoth {
		r.routePresencesFrom(contactJID, userJID, xml.UnavailableType)
	}
	return nil
}

func (r *ModRoster) processUnsubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.stm.JID()

	log.Infof("processing 'unsubscribed' - user: %s (%s/%s)", userJID, r.stm.Username(), r.stm.Resource())

	if err := r.deleteRosterNotification(userJID, contactJID); err != nil {
		return err
	}
	contactRi, err := rosterTable.fetchRosterItem(contactJID.Node(), userJID.Node())
	if err != nil {
		return err
	}
	contactSubscription := subscriptionNone
	if contactRi != nil {
		contactSubscription = contactRi.Subscription
		switch contactSubscription {
		case subscriptionBoth:
			contactRi.Subscription = subscriptionTo
		default:
			contactRi.Subscription = subscriptionNone
		}
		if err := rosterTable.insertOrUpdateRosterItem(contactRi); err != nil {
			return err
		}
		if err := r.pushRosterItem(contactRi, contactJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.UnsubscribedType)
	p.AppendElements(presence.Elements().All())

	if r.isLocalJID(userJID) {
		userRi, err := rosterTable.fetchRosterItem(userJID.Node(), contactJID.Node())
		if err != nil {
			return err
		}
		if userRi != nil {
			switch userRi.Subscription {
			case subscriptionBoth:
				userRi.Subscription = subscriptionFrom
			default:
				userRi.Subscription = subscriptionNone
			}
			userRi.Ask = false
			if err := rosterTable.insertOrUpdateRosterItem(userRi); err != nil {
				return err
			}
			if err := r.pushRosterItem(userRi, userJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, userJID)

	if contactSubscription == subscriptionFrom || contactSubscription == subscriptionBoth {
		r.routePresencesFrom(contactJID, userJID, xml.UnavailableType)
	}
	return nil
}

func (r *ModRoster) insertOrUpdateRosterNotification(userJID *xml.JID, contactJID *xml.JID, presence *xml.Presence) error {
	rn := &model.RosterNotification{
		User:     userJID.Node(),
		Contact:  contactJID.Node(),
		Elements: presence.Elements().All(),
	}
	return storage.Instance().InsertOrUpdateRosterNotification(rn)
}

func (r *ModRoster) deleteRosterNotification(userJID *xml.JID, contactJID *xml.JID) error {
	return storage.Instance().DeleteRosterNotification(userJID.Node(), contactJID.Node())
}

func (r *ModRoster) pushRosterItem(ri *model.RosterItem, to *xml.JID) error {
	query := xml.NewElementNamespace("query", rosterNamespace)
	query.AppendElement(r.elementFromRosterItem(ri))

	streams := c2s.Instance().AvailableStreams(to.Node())
	for _, strm := range streams {
		if !strm.IsRosterRequested() {
			continue
		}
		pushEl := xml.NewIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(strm.JID().String())
		pushEl.AppendElement(query)
		strm.SendElement(pushEl)
	}
	return nil
}

func (r *ModRoster) isLocalJID(jid *xml.JID) bool {
	return c2s.Instance().IsLocalDomain(jid.Domain())
}

func (r *ModRoster) routePresencesFrom(from *xml.JID, to *xml.JID, presenceType string) {
	fromStreams := c2s.Instance().AvailableStreams(from.Node())
	for _, fromStream := range fromStreams {
		p := xml.NewPresence(fromStream.JID(), to.ToBareJID(), presenceType)
		if presenceType == xml.AvailableType {
			p.AppendElements(fromStream.PresenceElements())
		}
		r.routePresence(p, to)
	}
}

func (r *ModRoster) routePresence(presence *xml.Presence, to *xml.JID) {
	if c2s.Instance().IsLocalDomain(to.Domain()) {
		toStreams := c2s.Instance().AvailableStreams(to.Node())
		for _, toStream := range toStreams {
			p := xml.NewPresence(presence.FromJID(), toStream.JID(), presence.Type())
			p.AppendElements(presence.Elements().All())
			toStream.SendElement(p)
		}
	} else {
		// TODO(ortuman): Implement XMPP federation
	}
}

func (r *ModRoster) rosterItemJID(ri *model.RosterItem) *xml.JID {
	j, _ := xml.NewJIDString(fmt.Sprintf("%s@%s", ri.Contact, r.stm.Domain()), true)
	return j
}

func (r *ModRoster) rosterItemFromElement(item xml.Element) (*model.RosterItem, error) {
	ri := &model.RosterItem{}
	if jid := item.Attributes().Get("jid"); len(jid) > 0 {
		j, err := xml.NewJIDString(jid, false)
		if err != nil {
			return nil, err
		}
		ri.Contact = j.Node()
	} else {
		return nil, errors.New("item 'jid' attribute is required")
	}
	ri.Name = item.Attributes().Get("name")

	subscription := item.Attributes().Get("subscription")
	if len(subscription) > 0 {
		switch subscription {
		case subscriptionBoth, subscriptionFrom, subscriptionTo, subscriptionNone, subscriptionRemove:
			break
		default:
			return nil, fmt.Errorf("unrecognized 'subscription' enum type: %s", subscription)
		}
		ri.Subscription = subscription
	}
	ask := item.Attributes().Get("ask")
	if len(ask) > 0 {
		if ask != "subscribe" {
			return nil, fmt.Errorf("unrecognized 'ask' enum type: %s", subscription)
		}
		ri.Ask = true
	}
	groups := item.Elements().Children("group")
	for _, group := range groups {
		if group.Attributes().Count() > 0 {
			return nil, errors.New("group element must not contain any attribute")
		}
		ri.Groups = append(ri.Groups, group.Text())
	}
	return ri, nil
}

func (r *ModRoster) elementFromRosterItem(ri *model.RosterItem) xml.Element {
	riJID := r.rosterItemJID(ri)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", riJID.ToBareJID().String())
	if len(ri.Name) > 0 {
		item.SetAttribute("name", ri.Name)
	}
	if len(ri.Subscription) > 0 {
		item.SetAttribute("subscription", ri.Subscription)
	}
	if ri.Ask {
		item.SetAttribute("ask", "subscribe")
	}
	for _, group := range ri.Groups {
		if len(group) == 0 {
			continue
		}
		gr := xml.NewElementName("group")
		gr.SetText(group)
		item.AppendElement(gr)
	}
	return item
}
