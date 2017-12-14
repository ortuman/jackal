/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"strings"

	"time"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/xml"
)

const privateStorageNamespace = "jabber:iq:private"

type XEPPrivateStorage struct {
	queue concurrent.OperationQueue
	strm  Stream
}

func NewXEPPrivateStorage(strm Stream) *XEPPrivateStorage {
	x := &XEPPrivateStorage{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second,
		},
		strm: strm,
	}
	return x
}

func (x *XEPPrivateStorage) AssociatedNamespaces() []string {
	return []string{privateStorageNamespace}
}

func (x *XEPPrivateStorage) MatchesIQ(iq *xml.IQ) bool {
	return iq.FindElementNamespace("query", privateStorageNamespace) != nil
}

func (x *XEPPrivateStorage) ProcessIQ(iq *xml.IQ) {
	x.queue.Async(func() {
		q := iq.FindElementNamespace("query", privateStorageNamespace)
		if q.ElementsCount() != 1 {
			x.strm.SendElement(iq.BadRequestError())
			return
		}
		toJid := iq.ToJID()
		validTo := toJid.IsServer() || toJid.Node() == x.strm.Username()
		if !validTo {
			x.strm.SendElement(iq.ForbiddenError())
			return
		}
		if iq.IsGet() {
			x.getPrivate(iq, q)
		} else if iq.IsSet() {
			x.setPrivate(iq, q)
		} else {
			x.strm.SendElement(iq.BadRequestError())
			return
		}
	})
}

func (x *XEPPrivateStorage) getPrivate(iq *xml.IQ, query *xml.Element) {
}

func (x *XEPPrivateStorage) setPrivate(iq *xml.IQ, query *xml.Element) {
}

func (x *XEPPrivateStorage) isValidNamespace(ns string) bool {
	return !strings.HasPrefix(ns, "jabber:") && !strings.HasPrefix(ns, "http://jabber.org/") && ns != "vcard-temp"
}
