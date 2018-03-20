/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const vCardNamespace = "vcard-temp"

// XEPVCard represents a vCard server stream module.
type XEPVCard struct {
	strm    c2s.Stream
	actorCh chan func()
	doneCh  chan struct{}
}

// NewXEPVCard returns a vCard IQ handler module.
func NewXEPVCard(strm c2s.Stream) *XEPVCard {
	v := &XEPVCard{
		strm:    strm,
		actorCh: make(chan func(), moduleMailboxSize),
		doneCh:  make(chan struct{}),
	}
	go v.actorLoop()
	return v
}

// AssociatedNamespaces returns namespaces associated
// with vCard module.
func (x *XEPVCard) AssociatedNamespaces() []string {
	return []string{vCardNamespace}
}

// Done signals stream termination.
func (x *XEPVCard) Done() {
	x.doneCh <- struct{}{}
}

// MatchesIQ returns whether or not an IQ should be
// processed by the vCard module.
func (x *XEPVCard) MatchesIQ(iq *xml.IQ) bool {
	return (iq.IsGet() || iq.IsSet()) && iq.FindElementNamespace("vCard", vCardNamespace) != nil
}

// ProcessIQ processes a vCard IQ taking according actions
// over the associated stream.
func (x *XEPVCard) ProcessIQ(iq *xml.IQ) {
	x.actorCh <- func() {
		vCard := iq.FindElementNamespace("vCard", vCardNamespace)
		if iq.IsGet() {
			x.getVCard(vCard, iq)
		} else if iq.IsSet() {
			x.setVCard(vCard, iq)
		}
	}
}

func (x *XEPVCard) actorLoop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.doneCh:
			return
		}
	}
}

func (x *XEPVCard) getVCard(vCard xml.Element, iq *xml.IQ) {
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
	log.Infof("retrieving vcard... (%s/%s)", x.strm.Username(), x.strm.Resource())

	resultIQ := iq.ResultIQ()
	if resElem != nil {
		resultIQ.AppendElement(resElem)
	} else {
		// empty vCard
		resultIQ.AppendElement(xml.NewElementNamespace("vCard", vCardNamespace))
	}
	x.strm.SendElement(resultIQ)
}

func (x *XEPVCard) setVCard(vCard xml.Element, iq *xml.IQ) {
	toJid := iq.ToJID()
	if toJid.IsServer() || (toJid.IsBare() && toJid.Node() == x.strm.Username()) {
		log.Infof("saving vcard... (%s/%s)", x.strm.Username(), x.strm.Resource())

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
