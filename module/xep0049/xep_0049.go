/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0049

import (
	"strings"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const privateStorageNamespace = "jabber:iq:private"

// XEPPrivateStorage represents a private storage server stream module.
type XEPPrivateStorage struct {
	stm     c2s.Stream
	actorCh chan func()
	doneCh  chan struct{}
}

// New returns a private storage IQ handler module.
func New(stm c2s.Stream) *XEPPrivateStorage {
	x := &XEPPrivateStorage{
		stm:     stm,
		actorCh: make(chan func(), 32),
		doneCh:  make(chan struct{}),
	}
	go x.actorLoop()
	return x
}

// AssociatedNamespaces returns namespaces associated
// with private storage module.
func (x *XEPPrivateStorage) AssociatedNamespaces() []string {
	return []string{}
}

// Done signals stream termination.
func (x *XEPPrivateStorage) Done() {
	x.doneCh <- struct{}{}
}

// MatchesIQ returns whether or not an IQ should be
// processed by the private storage module.
func (x *XEPPrivateStorage) MatchesIQ(iq *xml.IQ) bool {
	return iq.Elements().ChildNamespace("query", privateStorageNamespace) != nil
}

// ProcessIQ processes a private storage IQ taking according actions
// over the associated stream.
func (x *XEPPrivateStorage) ProcessIQ(iq *xml.IQ) {
	x.actorCh <- func() {
		q := iq.Elements().ChildNamespace("query", privateStorageNamespace)
		toJid := iq.ToJID()
		validTo := toJid.IsServer() || toJid.Node() == x.stm.Username()
		if !validTo {
			x.stm.SendElement(iq.ForbiddenError())
			return
		}
		if iq.IsGet() {
			x.getPrivate(iq, q)
		} else if iq.IsSet() {
			x.setPrivate(iq, q)
		} else {
			x.stm.SendElement(iq.BadRequestError())
			return
		}
	}
}

func (x *XEPPrivateStorage) actorLoop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.doneCh:
			return
		}
	}
}

func (x *XEPPrivateStorage) getPrivate(iq *xml.IQ, q xml.XElement) {
	if q.Elements().Count() != 1 {
		x.stm.SendElement(iq.NotAcceptableError())
		return
	}
	privElem := q.Elements().All()[0]
	privNS := privElem.Namespace()
	isValidNS := x.isValidNamespace(privNS)

	if privElem.Elements().Count() > 0 || !isValidNS {
		x.stm.SendElement(iq.NotAcceptableError())
		return
	}
	log.Infof("retrieving private element. ns: %s... (%s/%s)", privNS, x.stm.Username(), x.stm.Resource())

	privElements, err := storage.Instance().FetchPrivateXML(privNS, x.stm.Username())
	if err != nil {
		log.Errorf("%v", err)
		x.stm.SendElement(iq.InternalServerError())
		return
	}
	res := iq.ResultIQ()
	query := xml.NewElementNamespace("query", privateStorageNamespace)
	if privElements != nil {
		query.AppendElements(privElements)
	} else {
		query.AppendElement(xml.NewElementNamespace(privElem.Name(), privElem.Namespace()))
	}
	res.AppendElement(query)

	x.stm.SendElement(res)
}

func (x *XEPPrivateStorage) setPrivate(iq *xml.IQ, q xml.XElement) {
	nsElements := map[string][]xml.XElement{}

	for _, privElement := range q.Elements().All() {
		ns := privElement.Namespace()
		if len(ns) == 0 {
			x.stm.SendElement(iq.BadRequestError())
			return
		}
		if !x.isValidNamespace(privElement.Namespace()) {
			x.stm.SendElement(iq.NotAcceptableError())
			return
		}
		elems := nsElements[ns]
		if elems == nil {
			elems = []xml.XElement{privElement}
		} else {
			elems = append(elems, privElement)
		}
		nsElements[ns] = elems
	}
	for ns, elements := range nsElements {
		log.Infof("saving private element. ns: %s... (%s/%s)", ns, x.stm.Username(), x.stm.Resource())

		if err := storage.Instance().InsertOrUpdatePrivateXML(elements, ns, x.stm.Username()); err != nil {
			log.Errorf("%v", err)
			x.stm.SendElement(iq.InternalServerError())
			return
		}
	}
	x.stm.SendElement(iq.ResultIQ())
}

func (x *XEPPrivateStorage) isValidNamespace(ns string) bool {
	return !strings.HasPrefix(ns, "jabber:") && !strings.HasPrefix(ns, "http://jabber.org/") && ns != "vcard-temp"
}
