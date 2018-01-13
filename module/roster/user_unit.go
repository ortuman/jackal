/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
)

type userUnit struct {
	rosterUnit
	contactUnit *contactUnit
}

func (u *userUnit) updateRosterItem(ri *storage.RosterItem) {
}

func (u *userUnit) receiveUserPresence(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) {
	var err error
	switch presence.Type() {
	case xml.SubscribeType:
		err = u.userSubscribe(presence, userJID, contactJID)
	}
	if err != nil {
		log.Error(err)
	}
}

func (uu *userUnit) receiveContactPresence(presence *xml.Presence) {
}

func (uu *userUnit) userSubscribe(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) error {
	userRi, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return err
	}
	if userRi != nil {
		switch userRi.Subscription {
		case subscriptionTo, subscriptionBoth:
			return nil // already subscribed
		default:
			userRi.Ask = true
		}
	} else {
		// create roster item if not previously created
		userRi = &storage.RosterItem{
			Username:     userJID.Node(),
			JID:          contactJID,
			Subscription: subscriptionNone,
			Ask:          true,
		}
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(userRi); err != nil {
		return err
	}
	uu.pushRosterItem(userRi, userJID)

	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.SubscribeType)
	p.AppendElements(presence.Elements())
	uu.routePresence(p, userJID, contactJID)

	return nil
}

func (uu *userUnit) routePresence(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) {
	if stream.C2S().IsLocalDomain(contactJID.Domain()) {
		uu.contactUnit.receiveUserPresence(presence, userJID, contactJID)
	} else {
		// TODO(ortuman): Implement XMPP federation
	}
}
