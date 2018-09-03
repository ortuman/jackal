/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0054

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 2048

const vCardNamespace = "vcard-temp"

// VCard represents a vCard server stream module.
type VCard struct {
	actorCh    chan func()
	shutdownCh <-chan struct{}
}

// New returns a vCard IQ handler module.
func New(disco *xep0030.DiscoInfo, shutdownCh <-chan struct{}) *VCard {
	v := &VCard{
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: shutdownCh,
	}
	go v.loop()
	if disco != nil {
		disco.RegisterServerFeature(vCardNamespace)
		disco.RegisterAccountFeature(vCardNamespace)
	}
	return v
}

// MatchesIQ returns whether or not an IQ should be
// processed by the vCard module.
func (x *VCard) MatchesIQ(iq *xmpp.IQ) bool {
	return (iq.IsGet() || iq.IsSet()) && iq.Elements().ChildNamespace("vCard", vCardNamespace) != nil
}

// ProcessIQ processes a vCard IQ taking according actions
// over the associated stream.
func (x *VCard) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	x.actorCh <- func() { x.processIQ(iq, stm) }
}

// runs on it's own goroutine
func (x *VCard) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.shutdownCh:
			return
		}
	}
}

func (x *VCard) processIQ(iq *xmpp.IQ, stm stream.C2S) {
	vCard := iq.Elements().ChildNamespace("vCard", vCardNamespace)
	if vCard != nil {
		if iq.IsGet() {
			x.getVCard(vCard, iq, stm)
			return
		} else if iq.IsSet() {
			x.setVCard(vCard, iq, stm)
			return
		}
	}
	stm.SendElement(iq.BadRequestError())
}

func (x *VCard) getVCard(vCard xmpp.XElement, iq *xmpp.IQ, stm stream.C2S) {
	if vCard.Elements().Count() > 0 {
		stm.SendElement(iq.BadRequestError())
		return
	}
	toJID := iq.ToJID()
	resElem, err := storage.Instance().FetchVCard(toJID.Node())
	if err != nil {
		log.Errorf("%v", err)
		stm.SendElement(iq.InternalServerError())
		return
	}
	log.Infof("retrieving vcard... (%s/%s)", toJID.Node(), toJID.Resource())

	resultIQ := iq.ResultIQ()
	if resElem != nil {
		resultIQ.AppendElement(resElem)
	} else {
		// empty vCard
		resultIQ.AppendElement(xmpp.NewElementNamespace("vCard", vCardNamespace))
	}
	stm.SendElement(resultIQ)
}

func (x *VCard) setVCard(vCard xmpp.XElement, iq *xmpp.IQ, stm stream.C2S) {
	toJID := iq.ToJID()
	if toJID.IsServer() || (toJID.Node() == stm.Username()) {
		log.Infof("saving vcard... (%s/%s)", toJID.Node(), toJID.Resource())

		err := storage.Instance().InsertOrUpdateVCard(vCard, toJID.Node())
		if err != nil {
			log.Error(err)
			stm.SendElement(iq.InternalServerError())
			return

		}
		stm.SendElement(iq.ResultIQ())
	} else {
		stm.SendElement(iq.ForbiddenError())
	}
}
