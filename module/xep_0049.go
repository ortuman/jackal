/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
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

func NewXEPPrivateStorage(stream Stream) *XEPPrivateStorage {
	x := &XEPPrivateStorage{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second,
		},
		strm: stream,
	}
	return x
}

func (x *XEPPrivateStorage) AssociatedNamespaces() []string {
	return []string{}
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
	privNS := privElem.Namespace()
	isValidNS := x.isValidNamespace(privNS)

	if privElem.ElementsCount() > 0 || !isValidNS {
		x.strm.SendElement(iq.NotAcceptableError())
		return
	}
	log.Infof("retrieving private element. ns: %s... (%s/%s)", privNS, x.strm.Username(), x.strm.Resource())

	privElements, err := storage.Instance().FetchPrivateXML(privNS, x.strm.Username())
	if err != nil {
		log.Errorf("%v", err)
		x.strm.SendElement(iq.InternalServerError())
		return
	}
	res := iq.ResultIQ()
	query := xml.NewMutableElementNamespace("query", privateStorageNamespace)
	if privElements != nil {
		query.AppendElements(privElements)
	} else {
		query.AppendElement(xml.NewElementNamespace(privElem.Name(), privElem.Namespace()))
	}
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
		log.Infof("saving private element. ns: %s... (%s/%s)", ns, x.strm.Username(), x.strm.Resource())

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
