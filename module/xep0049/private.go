/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0049

import (
	"strings"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 2048

const privateNamespace = "jabber:iq:private"

// Private represents a private storage server stream module.
type Private struct {
	actorCh    chan func()
	shutdownCh chan chan bool
}

// New returns a private storage IQ handler module.
func New() (*Private, chan<- chan bool) {
	x := &Private{
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan bool),
	}
	go x.loop()
	return x, x.shutdownCh
}

// MatchesIQ returns whether or not an IQ should be
// processed by the private storage module.
func (x *Private) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.Elements().ChildNamespace("query", privateNamespace) != nil
}

// ProcessIQ processes a private storage IQ
// taking according actions over the associated stream
func (x *Private) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	x.actorCh <- func() { x.processIQ(iq, stm) }
}

// runs on it's own goroutine
func (x *Private) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case c := <-x.shutdownCh:
			c <- true
			return
		}
	}
}

func (x *Private) processIQ(iq *xmpp.IQ, stm stream.C2S) {
	q := iq.Elements().ChildNamespace("query", privateNamespace)
	toJid := iq.ToJID()
	validTo := toJid.IsServer() || toJid.Node() == stm.Username()
	if !validTo {
		stm.SendElement(iq.ForbiddenError())
		return
	}
	if iq.IsGet() {
		x.getPrivate(iq, q, stm)
	} else if iq.IsSet() {
		x.setPrivate(iq, q, stm)
	} else {
		stm.SendElement(iq.BadRequestError())
		return
	}
}

func (x *Private) getPrivate(iq *xmpp.IQ, q xmpp.XElement, stm stream.C2S) {
	if q.Elements().Count() != 1 {
		stm.SendElement(iq.NotAcceptableError())
		return
	}
	privElem := q.Elements().All()[0]
	privNS := privElem.Namespace()
	isValidNS := x.isValidNamespace(privNS)

	if privElem.Elements().Count() > 0 || !isValidNS {
		stm.SendElement(iq.NotAcceptableError())
		return
	}
	log.Infof("retrieving private element. ns: %s... (%s/%s)", privNS, stm.Username(), stm.Resource())

	privElements, err := storage.FetchPrivateXML(privNS, stm.Username())
	if err != nil {
		log.Error(err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	res := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", privateNamespace)
	if privElements != nil {
		query.AppendElements(privElements)
	} else {
		query.AppendElement(xmpp.NewElementNamespace(privElem.Name(), privElem.Namespace()))
	}
	res.AppendElement(query)

	stm.SendElement(res)
}

func (x *Private) setPrivate(iq *xmpp.IQ, q xmpp.XElement, stm stream.C2S) {
	nsElements := map[string][]xmpp.XElement{}

	for _, privElement := range q.Elements().All() {
		ns := privElement.Namespace()
		if len(ns) == 0 {
			stm.SendElement(iq.BadRequestError())
			return
		}
		if !x.isValidNamespace(privElement.Namespace()) {
			stm.SendElement(iq.NotAcceptableError())
			return
		}
		elems := nsElements[ns]
		if elems == nil {
			elems = []xmpp.XElement{privElement}
		} else {
			elems = append(elems, privElement)
		}
		nsElements[ns] = elems
	}
	for ns, elements := range nsElements {
		log.Infof("saving private element. ns: %s... (%s/%s)", ns, stm.Username(), stm.Resource())

		if err := storage.InsertOrUpdatePrivateXML(elements, ns, stm.Username()); err != nil {
			log.Error(err)
			stm.SendElement(iq.InternalServerError())
			return
		}
	}
	stm.SendElement(iq.ResultIQ())
}

func (x *Private) isValidNamespace(ns string) bool {
	return !strings.HasPrefix(ns, "jabber:") && !strings.HasPrefix(ns, "http://jabber.org/") && ns != "vcard-temp"
}
