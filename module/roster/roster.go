/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const rosterNamespace = "jabber:iq:roster"

// roster subscription values
const (
	SubscriptionNone   = "none"
	SubscriptionFrom   = "from"
	SubscriptionTo     = "to"
	SubscriptionBoth   = "both"
	SubscriptionRemove = "remove"
)

const (
	rosterRequestedContextKey = "roster:requested"
)

// Config represents roster module configuration.
type Config struct {
	Versioning bool `yaml:"versioning"`
}

// Roster represents a roster server stream module.
type Roster struct {
	cfg        *Config
	stm        router.C2S
	actorCh    chan func()
	errHandler func(error)
}

// New returns a roster server stream module.
func New(cfg *Config, stm router.C2S) *Roster {
	r := &Roster{
		cfg:        cfg,
		stm:        stm,
		actorCh:    make(chan func(), 32),
		errHandler: func(err error) { log.Error(err) },
	}
	go r.actorLoop(stm.Context().Done())
	return r
}

// AssociatedNamespaces returns namespaces associated
// with roster module.
func (r *Roster) AssociatedNamespaces() []string {
	return []string{}
}

// Done signals stream termination.
func (r *Roster) Done() {
}

// MatchesIQ returns whether or not an IQ should be
// processed by the roster module.
func (r *Roster) MatchesIQ(iq *xml.IQ) bool {
	return iq.Elements().ChildNamespace("query", rosterNamespace) != nil
}

// ProcessIQ processes a roster IQ taking according actions
// over the associated stream.
func (r *Roster) ProcessIQ(iq *xml.IQ) {
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

// ProcessPresence process an incoming roster presence.
func (r *Roster) ProcessPresence(presence *xml.Presence) {
	r.actorCh <- func() {
		if err := r.processPresence(presence); err != nil {
			r.errHandler(err)
		}
	}
}

// DeliverPendingApprovalNotifications delivers any pending roster notification
// to the associated stream.
func (r *Roster) DeliverPendingApprovalNotifications() {
	r.actorCh <- func() {
		if err := r.deliverPendingApprovalNotifications(); err != nil {
			r.errHandler(err)
		}
	}
}

// ReceivePresences delivers all inbound roster available presences
// to the associated module stream.
func (r *Roster) ReceivePresences() {
	r.actorCh <- func() {
		if err := r.receivePresences(); err != nil {
			r.errHandler(err)
		}
	}
}

// BroadcastPresence broadcasts presence to all outbound roster contacts.
func (r *Roster) BroadcastPresence(presence *xml.Presence) {
	r.actorCh <- func() {
		if err := r.broadcastPresence(presence); err != nil {
			r.errHandler(err)
		}
	}
}

// BroadcastPresenceAndWait broadcasts presence to all outbound
// roster contacts in a synchronous manner.
func (r *Roster) BroadcastPresenceAndWait(presence *xml.Presence) {
	continueCh := make(chan struct{})
	r.actorCh <- func() {
		if err := r.broadcastPresence(presence); err != nil {
			r.errHandler(err)
		}
		close(continueCh)
	}
	<-continueCh
}

func (r *Roster) actorLoop(doneCh <-chan struct{}) {
	for {
		select {
		case f := <-r.actorCh:
			f()
		case <-doneCh:
			return
		}
	}
}

func (r *Roster) processPresence(presence *xml.Presence) error {
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

func (r *Roster) deliverPendingApprovalNotifications() error {
	rns, err := storage.Instance().FetchRosterNotifications(r.stm.Username())
	if err != nil {
		return err
	}
	for _, rn := range rns {
		fromJID, _ := xml.NewJIDString(rn.JID, true)
		p := xml.NewPresence(fromJID, r.stm.JID(), xml.SubscribeType)
		p.AppendElements(rn.Elements)
		r.stm.SendElement(p)
	}
	return nil
}

func (r *Roster) receivePresences() error {
	items, _, err := storage.Instance().FetchRosterItems(r.stm.Username())
	if err != nil {
		return err
	}
	usrJID := r.stm.JID()
	for _, item := range items {
		switch item.Subscription {
		case SubscriptionTo, SubscriptionBoth:
			r.routePresencesFrom(r.rosterItemJID(&item), usrJID, xml.AvailableType)
		}
	}
	return nil
}

func (r *Roster) broadcastPresence(presence *xml.Presence) error {
	itms, _, err := storage.Instance().FetchRosterItems(r.stm.Username())
	if err != nil {
		return err
	}
	for _, itm := range itms {
		switch itm.Subscription {
		case SubscriptionFrom, SubscriptionBoth:
			p := xml.NewPresence(r.stm.JID(), r.rosterItemJID(&itm), presence.Type())
			p.AppendElements(p.Elements().All())
			router.Instance().Route(p)
		}
	}
	return nil
}

func (r *Roster) sendRoster(iq *xml.IQ, query xml.XElement) {
	if query.Elements().Count() > 0 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("retrieving user roster... (%s/%s)", r.stm.Username(), r.stm.Resource())

	itms, ver, err := storage.Instance().FetchRosterItems(r.stm.Username())
	if err != nil {
		r.errHandler(err)
		r.stm.SendElement(iq.InternalServerError())
		return
	}
	v := r.parseVer(query.Attributes().Get("ver"))

	res := iq.ResultIQ()
	if !r.cfg.Versioning || v == 0 || v < ver.DeletionVer {
		// push all roster items
		q := xml.NewElementNamespace("query", rosterNamespace)
		if r.cfg.Versioning {
			q.SetAttribute("ver", fmt.Sprintf("v%d", ver.Ver))
		}
		for _, itm := range itms {
			q.AppendElement(r.elementFromRosterItem(&itm))
		}
		res.AppendElement(q)
		r.stm.SendElement(res)
	} else {
		// push roster changes
		r.stm.SendElement(res)
		for _, itm := range itms {
			if itm.Ver > v {
				iq := xml.NewIQType(uuid.New(), xml.SetType)
				q := xml.NewElementNamespace("query", rosterNamespace)
				q.SetAttribute("ver", fmt.Sprintf("v%d", itm.Ver))
				q.AppendElement(r.elementFromRosterItem(&itm))
				iq.AppendElement(q)
				r.stm.SendElement(iq)
			}
		}
	}
	r.stm.Context().SetBool(true, rosterRequestedContextKey)
}

func (r *Roster) updateRoster(iq *xml.IQ, query xml.XElement) {
	itms := query.Elements().Children("item")
	if len(itms) != 1 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	ri, err := r.rosterItemFromElement(itms[0])
	if err != nil {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	switch ri.Subscription {
	case SubscriptionRemove:
		if err := r.removeItem(ri); err != nil {
			r.errHandler(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	default:
		if err := r.updateItem(ri); err != nil {
			r.errHandler(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	}
	r.stm.SendElement(iq.ResultIQ())
}

func (r *Roster) removeItem(ri *model.RosterItem) error {
	var unsubscribe, unsubscribed *xml.Presence

	usrJID := r.stm.JID().ToBareJID()
	cntJID := r.rosterItemJID(ri).ToBareJID()

	log.Infof("removing roster item: %v (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.String())
	if err != nil {
		return err
	}
	usrSub := SubscriptionNone
	if usrRi != nil {
		usrSub = usrRi.Subscription
		switch usrSub {
		case SubscriptionTo:
			unsubscribe = xml.NewPresence(usrJID, cntJID, xml.UnsubscribeType)
		case SubscriptionFrom:
			unsubscribed = xml.NewPresence(usrJID, cntJID, xml.UnsubscribedType)
		case SubscriptionBoth:
			unsubscribe = xml.NewPresence(usrJID, cntJID, xml.UnsubscribeType)
			unsubscribed = xml.NewPresence(usrJID, cntJID, xml.UnsubscribedType)
		}
		usrRi.Subscription = SubscriptionRemove
		usrRi.Ask = false

		if err := r.deleteNotification(cntJID.Node(), usrJID); err != nil {
			return err
		}
		if err := r.deleteItem(usrRi, usrJID); err != nil {
			return err
		}
	}

	if router.Instance().IsLocalDomain(cntJID.Domain()) {
		cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			if cntRi.Subscription == SubscriptionFrom || cntRi.Subscription == SubscriptionBoth {
				r.routePresencesFrom(cntJID, usrJID, xml.UnavailableType)
			}
			switch cntRi.Subscription {
			case SubscriptionBoth:
				cntRi.Subscription = SubscriptionTo
				if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
					return err
				}
				fallthrough

			default:
				cntRi.Subscription = SubscriptionNone
				if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
					return err
				}
			}
		}
	}
	if unsubscribe != nil {
		router.Instance().Route(unsubscribe)
	}
	if unsubscribed != nil {
		router.Instance().Route(unsubscribed)
	}

	if usrSub == SubscriptionFrom || usrSub == SubscriptionBoth {
		r.routePresencesFrom(usrJID, cntJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) updateItem(ri *model.RosterItem) error {
	usrJID := r.stm.JID().ToBareJID()
	cntJID := r.rosterItemJID(ri).ToBareJID()

	log.Infof("updating roster item - contact: %s (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.String())
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
		usrRi = &model.RosterItem{
			Username:     r.stm.Username(),
			JID:          ri.JID,
			Name:         ri.Name,
			Subscription: SubscriptionNone,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	return r.insertOrUpdateItem(usrRi, r.stm.JID())
}

func (r *Roster) processSubscribe(presence *xml.Presence) error {
	usrJID := r.stm.JID().ToBareJID()
	cntJID := presence.ToJID().ToBareJID()

	log.Infof("processing 'subscribe' - contact: %s (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.String())
	if err != nil {
		return err
	}
	if usrRi != nil {
		switch usrRi.Subscription {
		case SubscriptionTo, SubscriptionBoth:
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
		usrRi = &model.RosterItem{
			Username:     usrJID.Node(),
			JID:          cntJID.String(),
			Subscription: SubscriptionNone,
			Ask:          true,
		}
	}
	if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
		return err
	}
	// stamp the presence stanza of type "subscribe" with the user's bare JID as the 'from' address
	p := xml.NewPresence(usrJID, cntJID, xml.SubscribeType)
	p.AppendElements(presence.Elements().All())

	if router.Instance().IsLocalDomain(cntJID.Domain()) {
		// archive roster approval notification
		if err := r.insertOrUpdateNotification(cntJID.Node(), usrJID, p); err != nil {
			return err
		}
	}
	router.Instance().Route(p)
	return nil
}

func (r *Roster) processSubscribed(presence *xml.Presence) error {
	usrJID := presence.ToJID().ToBareJID()
	cntJID := r.stm.JID().ToBareJID()

	log.Infof("processing 'subscribed' - user: %s (%s/%s)", usrJID, r.stm.Username(), r.stm.Resource())

	if err := r.deleteNotification(cntJID.Node(), usrJID); err != nil {
		return err
	}
	cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.String())
	if err != nil {
		return err
	}
	if cntRi != nil {
		switch cntRi.Subscription {
		case SubscriptionTo:
			cntRi.Subscription = SubscriptionBoth
		case SubscriptionNone:
			cntRi.Subscription = SubscriptionFrom
		}
	} else {
		// create roster item if not previously created
		cntRi = &model.RosterItem{
			Username:     cntJID.Node(),
			JID:          usrJID.String(),
			Subscription: SubscriptionFrom,
			Ask:          false,
		}
	}
	if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
		return err
	}
	// stamp the presence stanza of type "subscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(cntJID, usrJID, xml.SubscribedType)
	p.AppendElements(presence.Elements().All())

	if router.Instance().IsLocalDomain(usrJID.Domain()) {
		usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case SubscriptionFrom:
				usrRi.Subscription = SubscriptionBoth
			case SubscriptionNone:
				usrRi.Subscription = SubscriptionTo
			default:
				return nil
			}
			usrRi.Ask = false
			if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
				return err
			}
		}
	}
	router.Instance().Route(p)
	r.routePresencesFrom(cntJID, usrJID, xml.AvailableType)
	return nil
}

func (r *Roster) processUnsubscribe(presence *xml.Presence) error {
	usrJID := r.stm.JID().ToBareJID()
	cntJID := presence.ToJID().ToBareJID()

	log.Infof("processing 'unsubscribe' - contact: %s (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.String())
	if err != nil {
		return err
	}
	usrSub := SubscriptionNone
	if usrRi != nil {
		usrSub = usrRi.Subscription
		switch usrSub {
		case SubscriptionBoth:
			usrRi.Subscription = SubscriptionFrom
		default:
			usrRi.Subscription = SubscriptionNone
		}
		if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribe" with the users's bare JID as the 'from' address
	p := xml.NewPresence(usrJID, cntJID, xml.UnsubscribeType)
	p.AppendElements(presence.Elements().All())

	if router.Instance().IsLocalDomain(cntJID.Domain()) {
		cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			switch cntRi.Subscription {
			case SubscriptionBoth:
				cntRi.Subscription = SubscriptionTo
			default:
				cntRi.Subscription = SubscriptionNone
			}
			if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
				return err
			}
		}
	}
	router.Instance().Route(p)

	if usrSub == SubscriptionTo || usrSub == SubscriptionBoth {
		r.routePresencesFrom(cntJID, usrJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) processUnsubscribed(presence *xml.Presence) error {
	usrJID := presence.ToJID().ToBareJID()
	cntJID := r.stm.JID().ToBareJID()

	log.Infof("processing 'unsubscribed' - user: %s (%s/%s)", usrJID, r.stm.Username(), r.stm.Resource())

	if err := r.deleteNotification(cntJID.Node(), usrJID); err != nil {
		return err
	}
	cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.String())
	if err != nil {
		return err
	}
	cntSub := SubscriptionNone
	if cntRi != nil {
		cntSub = cntRi.Subscription
		switch cntSub {
		case SubscriptionBoth:
			cntRi.Subscription = SubscriptionTo
		default:
			cntRi.Subscription = SubscriptionNone
		}
		if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(cntJID, usrJID, xml.UnsubscribedType)
	p.AppendElements(presence.Elements().All())

	if router.Instance().IsLocalDomain(usrJID.Domain()) {
		usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.String())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case SubscriptionBoth:
				usrRi.Subscription = SubscriptionFrom
			default:
				usrRi.Subscription = SubscriptionNone
			}
			usrRi.Ask = false
			if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
				return err
			}
		}
	}
	router.Instance().Route(p)

	if cntSub == SubscriptionFrom || cntSub == SubscriptionBoth {
		r.routePresencesFrom(cntJID, usrJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) insertOrUpdateNotification(contact string, userJID *xml.JID, presence *xml.Presence) error {
	rn := &model.RosterNotification{
		Contact:  contact,
		JID:      userJID.String(),
		Elements: presence.Elements().All(),
	}
	return storage.Instance().InsertOrUpdateRosterNotification(rn)
}

func (r *Roster) deleteNotification(contact string, userJID *xml.JID) error {
	return storage.Instance().DeleteRosterNotification(contact, userJID.String())
}

func (r *Roster) insertOrUpdateItem(ri *model.RosterItem, pushTo *xml.JID) error {
	v, err := storage.Instance().InsertOrUpdateRosterItem(ri)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return r.pushItem(ri, pushTo)
}

func (r *Roster) deleteItem(ri *model.RosterItem, pushTo *xml.JID) error {
	v, err := storage.Instance().DeleteRosterItem(ri.Username, ri.JID)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return r.pushItem(ri, pushTo)
}

func (r *Roster) pushItem(ri *model.RosterItem, to *xml.JID) error {
	query := xml.NewElementNamespace("query", rosterNamespace)
	if r.cfg.Versioning {
		query.SetAttribute("ver", fmt.Sprintf("v%d", ri.Ver))
	}
	query.AppendElement(r.elementFromRosterItem(ri))

	stms := router.Instance().StreamsMatchingJID(to.ToBareJID())
	for _, stm := range stms {
		if !stm.Context().Bool(rosterRequestedContextKey) {
			continue
		}
		pushEl := xml.NewIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(stm.JID().String())
		pushEl.AppendElement(query)
		stm.SendElement(pushEl)
	}
	return nil
}

func (r *Roster) routePresencesFrom(from *xml.JID, to *xml.JID, presenceType string) {
	stms := router.Instance().StreamsMatchingJID(from.ToBareJID())
	for _, stm := range stms {
		p := xml.NewPresence(stm.JID(), to.ToBareJID(), presenceType)
		if presence := stm.Presence(); presence != nil && presenceType == xml.AvailableType {
			p.AppendElements(presence.Elements().All())
		}
		router.Instance().Route(p)
	}
}

func (r *Roster) rosterItemJID(ri *model.RosterItem) *xml.JID {
	j, _ := xml.NewJIDString(ri.JID, true)
	return j
}

func (r *Roster) rosterItemFromElement(item xml.XElement) (*model.RosterItem, error) {
	ri := &model.RosterItem{}
	if jid := item.Attributes().Get("jid"); len(jid) > 0 {
		j, err := xml.NewJIDString(jid, false)
		if err != nil {
			return nil, err
		}
		ri.JID = j.String()
	} else {
		return nil, errors.New("item 'jid' attribute is required")
	}
	ri.Name = item.Attributes().Get("name")

	subscription := item.Attributes().Get("subscription")
	if len(subscription) > 0 {
		switch subscription {
		case SubscriptionBoth, SubscriptionFrom, SubscriptionTo, SubscriptionNone, SubscriptionRemove:
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

func (r *Roster) elementFromRosterItem(ri *model.RosterItem) xml.XElement {
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

func (r *Roster) parseVer(ver string) int {
	if len(ver) > 0 && ver[0] == 'v' {
		v, _ := strconv.Atoi(ver[1:])
		return v
	}
	return 0
}
