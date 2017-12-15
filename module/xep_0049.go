/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"strings"

	"time"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
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

func (x *XEPPrivateStorage) getPrivate(iq *xml.IQ, q *xml.Element) {
	if q.ElementsCount() != 1 {
		x.strm.SendElement(iq.NotAcceptableError())
		return
	}
	privElem := q.Elements()[0]
	isValidNS := x.isValidNamespace(privElem.Namespace())

	if privElem.ElementsCount() > 0 || !isValidNS {
		x.strm.SendElement(iq.NotAcceptableError())
		return
	}
	privElements, err := storage.Instance().FetchPrivateXML(privElem.Namespace(), x.strm.Username())
	if err != nil {
		log.Errorf("%v", err)
		x.strm.SendElement(iq.InternalServerError())
		return
	}
	res := iq.ResultIQ()
	query := xml.NewMutableElementNamespace("query", privateStorageNamespace)
	query.AppendElements(privElements)
	res.AppendElement(query.Copy())

	x.strm.SendElement(res)
}

func (x *XEPPrivateStorage) setPrivate(iq *xml.IQ, q *xml.Element) {
	nsElements := map[string][]*xml.Element{}

	for _, privElement := range q.Elements() {
		ns := privElement.Namespace()
		if len(ns) == 0 {
			x.strm.SendElement(iq.BadRequestError())
			return
		}
		if !x.isValidNamespace(privElement.Namespace()) {
			x.strm.SendElement(iq.NotAcceptableError())
			return
		}
		elems := nsElements[ns]
		if elems == nil {
			elems = []*xml.Element{privElement}
		} else {
			elems = append(elems, privElement)
		}
		nsElements[ns] = elems
	}
	for ns, elements := range nsElements {
		if err := storage.Instance().InsertOrUpdatePrivateXML(elements, ns, x.strm.Username()); err != nil {
			log.Errorf("%v", err)
			x.strm.SendElement(iq.InternalServerError())
			return
		}
	}
	x.strm.SendElement(iq.ResultIQ())
}

func (x *XEPPrivateStorage) isValidNamespace(ns string) bool {
	return !strings.HasPrefix(ns, "jabber:") && !strings.HasPrefix(ns, "http://jabber.org/") && ns != "vcard-temp"
}