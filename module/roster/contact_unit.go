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

type contactUnit struct {
	rosterUnit
	userUnit *userUnit
}

func (cu *contactUnit) processPresence(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) {
	var err error
	switch presence.Type() {
	case xml.SubscribeType:
		err = cu.contactSubscribe(presence, userJID, contactJID)
	case xml.SubscribedType:
		err = cu.contactSubscribed(presence, userJID, contactJID)
	}
	if err != nil {
		log.Error(err)
	}
}

func (cu *contactUnit) contactSubscribe(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) error {
	// archive roster approval notification
	err := storage.Instance().InsertOrUpdateRosterNotification(userJID.Node(), contactJID.ToBareJID(), presence)
	if err != nil {
		return err
	}
	cu.broadcastPresence(presence, contactJID)
	return nil
}

func (cu *contactUnit) contactSubscribed(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) error {
	contactRi, err := storage.Instance().FetchRosterItem(contactJID.Node(), userJID.ToBareJID())
	if err != nil {
		return err
	}
	if contactRi == nil {
		return nil
	}
	// update contact's roster item...
	switch contactRi.Subscription {
	case subscriptionTo:
		contactRi.Subscription = subscriptionBoth
	case subscriptionNone:
		contactRi.Subscription = subscriptionFrom
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(contactRi); err != nil {
		return err
	}
	cu.pushRosterItem(contactRi, contactJID)

	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.SubscribedType)
	p.AppendElements(presence.Elements())
	cu.routePresence(p, userJID, contactJID)
	return nil
}

func (cu *contactUnit) routePresence(presence *xml.Presence, userJID *xml.JID, contactJID *xml.JID) {
	if stream.C2S().IsLocalDomain(contactJID.Domain()) {
		cu.userUnit.processPresence(presence, userJID, contactJID)
	} else {
		// TODO(ortuman): Implement XMPP federation
	}
}
