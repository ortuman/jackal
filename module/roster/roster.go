/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"fmt"
	"strconv"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const rosterNamespace = "jabber:iq:roster"

// Config represents a roster configuration.
type Config struct {
	Versioning bool `yaml:"versioning"`
}

// Roster represents a roster server stream module.
type Roster struct {
	cfg     *Config
	ph      *PresenceHandler
	stm     stream.C2S
	actorCh chan func()
}

// New returns a roster server stream module.
func New(cfg *Config, stm stream.C2S) *Roster {
	r := &Roster{
		cfg:     cfg,
		ph:      NewPresenceHandler(cfg),
		stm:     stm,
		actorCh: make(chan func(), 32),
	}
	go r.actorLoop(stm.Context().Done())
	return r
}

// RegisterDisco registers disco entity features/items
// associated to roster module.
func (r *Roster) RegisterDisco(_ *xep0030.DiscoInfo) {
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
	doneCh := make(chan struct{})
	r.actorCh <- func() {
		if err := r.ph.ProcessPresence(presence); err != nil {
			log.Error(err)
		}
		close(doneCh)
	}
	<-doneCh
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

func (r *Roster) sendRoster(iq *xml.IQ, query xml.XElement) {
	if query.Elements().Count() > 0 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	userJID := r.stm.JID()

	log.Infof("retrieving user roster... (%s)", userJID)

	itms, ver, err := storage.Instance().FetchRosterItems(userJID.Node())
	if err != nil {
		log.Error(err)
		r.stm.SendElement(iq.InternalServerError())
		return
	}
	v := r.parseVer(query.Attributes().Get("ver"))

	res := iq.ResultIQ()
	if v == 0 || v < ver.DeletionVer {
		// push all roster items
		q := xml.NewElementNamespace("query", rosterNamespace)
		if r.cfg.Versioning {
			q.SetAttribute("ver", fmt.Sprintf("v%d", ver.Ver))
		}
		for _, itm := range itms {
			q.AppendElement(itm.Element())
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
				q.AppendElement(itm.Element())
				iq.AppendElement(q)
				r.stm.SendElement(iq)
			}
		}
	}
	r.stm.Context().SetBool(true, rosterRequestedCtxKey)
}

func (r *Roster) updateRoster(iq *xml.IQ, query xml.XElement) {
	itms := query.Elements().Children("item")
	if len(itms) != 1 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	ri, err := rostermodel.NewItem(itms[0])
	if err != nil {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	switch ri.Subscription {
	case rostermodel.SubscriptionRemove:
		if err := r.removeItem(ri); err != nil {
			log.Error(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	default:
		if err := r.updateItem(ri); err != nil {
			log.Error(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	}
	r.stm.SendElement(iq.ResultIQ())
}

func (r *Roster) updateItem(ri *rostermodel.Item) error {
	userJID := r.stm.JID().ToBareJID()
	contactJID := ri.ContactJID()

	log.Infof("updating roster item - contact: %s (%s)", contactJID, userJID)

	usrRi, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.String())
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
	return insertItem(usrRi, userJID, r.cfg.Versioning)
}

func (r *Roster) removeItem(ri *rostermodel.Item) error {
	var unsubscribe, unsubscribed *xml.Presence

	userJID := r.stm.JID().ToBareJID()
	contactJID := ri.ContactJID()

	log.Infof("removing roster item: %v (%s)", contactJID, userJID)

	usrRi, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.String())
	if err != nil {
		return err
	}
	usrSub := rostermodel.SubscriptionNone
	if usrRi != nil {
		usrSub = usrRi.Subscription
		switch usrSub {
		case rostermodel.SubscriptionTo:
			unsubscribe = xml.NewPresence(userJID, contactJID, xml.UnsubscribeType)
		case rostermodel.SubscriptionFrom:
			unsubscribed = xml.NewPresence(userJID, contactJID, xml.UnsubscribedType)
		case rostermodel.SubscriptionBoth:
			unsubscribe = xml.NewPresence(userJID, contactJID, xml.UnsubscribeType)
			unsubscribed = xml.NewPresence(userJID, contactJID, xml.UnsubscribedType)
		}
		usrRi.Subscription = rostermodel.SubscriptionRemove
		usrRi.Ask = false

		_, err := deleteNotification(contactJID.Node(), userJID)
		if err != nil {
			return err
		}
		if err := deleteItem(usrRi, userJID, r.cfg.Versioning); err != nil {
			return err
		}
	}
	if host.IsLocalHost(contactJID.Domain()) {
		cntRi, err := storage.Instance().FetchRosterItem(contactJID.Node(), userJID.String())
		if err != nil {
			return err
		}
		if cntRi != nil {
			if cntRi.Subscription == rostermodel.SubscriptionFrom || cntRi.Subscription == rostermodel.SubscriptionBoth {
				routePresencesFrom(contactJID, userJID, xml.UnavailableType)
			}
			switch cntRi.Subscription {
			case rostermodel.SubscriptionBoth:
				cntRi.Subscription = rostermodel.SubscriptionTo
				if insertItem(cntRi, contactJID, r.cfg.Versioning); err != nil {
					return err
				}
				fallthrough

			default:
				cntRi.Subscription = rostermodel.SubscriptionNone
				if insertItem(cntRi, contactJID, r.cfg.Versioning); err != nil {
					return err
				}
			}
		}
	}
	if unsubscribe != nil {
		router.Route(unsubscribe)
	}
	if unsubscribed != nil {
		router.Route(unsubscribed)
	}

	if usrSub == rostermodel.SubscriptionFrom || usrSub == rostermodel.SubscriptionBoth {
		routePresencesFrom(userJID, contactJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) parseVer(ver string) int {
	if len(ver) > 0 && ver[0] == 'v' {
		v, _ := strconv.Atoi(ver[1:])
		return v
	}
	return 0
}
