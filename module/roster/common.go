/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"fmt"

	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
)

const (
	rosterRequestedCtxKey = "roster:requested"
)

func insertItem(ri *rostermodel.Item, pushTo *jid.JID, versioning bool) error {
	v, err := storage.Instance().InsertOrUpdateRosterItem(ri)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return pushItem(ri, pushTo, versioning)
}

func deleteItem(ri *rostermodel.Item, pushTo *jid.JID, versioning bool) error {
	v, err := storage.Instance().DeleteRosterItem(ri.Username, ri.JID)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return pushItem(ri, pushTo, versioning)
}

func pushItem(ri *rostermodel.Item, to *jid.JID, versioning bool) error {
	query := xml.NewElementNamespace("query", rosterNamespace)
	if versioning {
		query.SetAttribute("ver", fmt.Sprintf("v%d", ri.Ver))
	}
	query.AppendElement(ri.Element())

	stms := router.UserStreams(to.Node())
	for _, stm := range stms {
		if !stm.Context().Bool(rosterRequestedCtxKey) {
			continue
		}
		pushEl := xml.NewIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(stm.JID().String())
		pushEl.AppendElement(query)
		stm.SendElement(pushEl)
	}
	return nil
}

func deleteNotification(contact string, userJID *jid.JID) (deleted bool, err error) {
	rn, err := storage.Instance().FetchRosterNotification(contact, userJID.String())
	if err != nil {
		return false, err
	}
	if rn == nil {
		return false, nil
	}
	if err := storage.Instance().DeleteRosterNotification(contact, userJID.String()); err != nil {
		return false, err
	}
	return true, nil
}

func insertOrUpdateNotification(contact string, userJID *jid.JID, presence *xml.Presence) error {
	rn := &rostermodel.Notification{
		Contact:  contact,
		JID:      userJID.String(),
		Presence: presence,
	}
	return storage.Instance().InsertOrUpdateRosterNotification(rn)
}

func routePresencesFrom(from *jid.JID, to *jid.JID, presenceType string) {
	stms := router.UserStreams(from.Node())
	for _, stm := range stms {
		p := xml.NewPresence(stm.JID(), to.ToBareJID(), presenceType)
		if presence := stm.Presence(); presence != nil && presence.IsAvailable() {
			p.AppendElements(presence.Elements().All())
		}
		router.Route(p)
	}
}
