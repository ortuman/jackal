/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"time"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xml"
)

const vCardNamespace = "vcard-temp"

type XEPVCard struct {
	queue concurrent.OperationQueue
	strm  Stream
}

func NewXEPVCard(strm Stream) *XEPVCard {
	v := &XEPVCard{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second,
		},
		strm: strm,
	}
	return v
}

func (x *XEPVCard) AssociatedNamespaces() []string {
	return []string{vCardNamespace}
}

func (x *XEPVCard) MatchesIQ(iq *xml.IQ) bool {
	return (iq.IsGet() || iq.IsSet()) && iq.FindElementNamespace("vCard", vCardNamespace) != nil
}

func (x *XEPVCard) ProcessIQ(iq *xml.IQ) {
	x.queue.Async(func() {
		vCard := iq.FindElementNamespace("vCard", vCardNamespace)
		if iq.IsGet() {
			x.getVCard(vCard, iq)
		} else if iq.IsSet() {
			x.setVCard(vCard, iq)
		}
	})
}

func (x *XEPVCard) getVCard(vCard *xml.Element, iq *xml.IQ) {
	if vCard.ElementsCount() > 0 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	toJid := iq.ToJID()

	var username string
	if toJid.IsServer() {
		username = x.strm.Username()
	} else {
		username = toJid.Node()
	}

	resElem, err := storage.Instance().FetchVCard(username)
	if err != nil {
		log.Errorf("%v", err)
		x.strm.SendElement(iq.InternalServerError())
		return
	}

	resultIQ := iq.ResultIQ()
	if resElem != nil {
		resultIQ.AppendElement(resElem)
	} else {
		// empty vCard
		resultIQ.AppendElement(xml.NewElementNamespace("vCard", vCardNamespace))
	}
	x.strm.SendElement(resultIQ)
}

func (x *XEPVCard) setVCard(vCard *xml.Element, iq *xml.IQ) {
	toJid := iq.ToJID()
	if toJid.IsServer() || (toJid.IsBare() && toJid.Node() == x.strm.Username()) {
		err := storage.Instance().InsertOrUpdateVCard(vCard, x.strm.Username())
		if err != nil {
			log.Errorf("%v", err)
			x.strm.SendElement(iq.InternalServerError())
			return
		}
		x.strm.SendElement(iq.ResultIQ())
	} else {
		x.strm.SendElement(iq.ForbiddenError())
	}
}
