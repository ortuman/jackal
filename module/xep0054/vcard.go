/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0054

import (
	"context"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
)

const vCardNamespace = "vcard-temp"

// VCard represents a vCard server stream module.
type VCard struct {
	router   *router.Router
	runQueue *runqueue.RunQueue
	rep      repository.VCard
}

// New returns a vCard IQ handler module.
func New(disco *xep0030.DiscoInfo, router *router.Router, rep repository.VCard) *VCard {
	v := &VCard{
		router:   router,
		runQueue: runqueue.New("xep0054"),
		rep:      rep,
	}
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

// ProcessIQ processes a vCard IQ taking according actions over the associated stream.
func (x *VCard) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq)
	})
}

// Shutdown shuts down vCard module.
func (x *VCard) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *VCard) processIQ(ctx context.Context, iq *xmpp.IQ) {
	vCard := iq.Elements().ChildNamespace("vCard", vCardNamespace)
	if vCard != nil {
		if iq.IsGet() {
			x.getVCard(ctx, vCard, iq)
			return
		} else if iq.IsSet() {
			x.setVCard(ctx, vCard, iq)
			return
		}
	}
	_ = x.router.Route(ctx, iq.BadRequestError())
}

func (x *VCard) getVCard(ctx context.Context, vCard xmpp.XElement, iq *xmpp.IQ) {
	if vCard.Elements().Count() > 0 {
		_ = x.router.Route(ctx, iq.BadRequestError())
		return
	}
	toJID := iq.ToJID()
	resElem, err := x.rep.FetchVCard(ctx, toJID.Node())
	if err != nil {
		log.Errorf("%v", err)
		_ = x.router.Route(ctx, iq.InternalServerError())
		return
	}
	log.Infof("retrieving vcard... (jid: %s)", toJID.String())

	resultIQ := iq.ResultIQ()
	if resElem != nil {
		resultIQ.AppendElement(resElem)
	} else {
		// empty vCard
		resultIQ.AppendElement(xmpp.NewElementNamespace("vCard", vCardNamespace))
	}
	_ = x.router.Route(ctx, resultIQ)
}

func (x *VCard) setVCard(ctx context.Context, vCard xmpp.XElement, iq *xmpp.IQ) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if toJID.IsServer() || (toJID.Node() == fromJID.Node()) {
		log.Infof("saving vcard... (jid: %s)", toJID.String())

		err := x.rep.UpsertVCard(ctx, vCard, toJID.Node())
		if err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return

		}
		_ = x.router.Route(ctx, iq.ResultIQ())
	} else {
		_ = x.router.Route(ctx, iq.ForbiddenError())
	}
}
