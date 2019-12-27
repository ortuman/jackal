/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0049

import (
	"context"
	"strings"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
)

const privateNamespace = "jabber:iq:private"

// Private represents a private storage server stream module.
type Private struct {
	router   *router.Router
	runQueue *runqueue.RunQueue
}

// New returns a private storage IQ handler module.
func New(router *router.Router) *Private {
	x := &Private{
		router:   router,
		runQueue: runqueue.New("xep0049"),
	}
	return x
}

// MatchesIQ returns whether or not an IQ should be processed by the private storage module.
func (x *Private) MatchesIQ(iq *xmpp.IQ) bool {
	return iq.Elements().ChildNamespace("query", privateNamespace) != nil
}

// ProcessIQ processes a private storage IQ taking according actions over the associated stream.
func (x *Private) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq)
	})
}

// Shutdown shuts down private storage module.
func (x *Private) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *Private) processIQ(ctx context.Context, iq *xmpp.IQ) {
	q := iq.Elements().ChildNamespace("query", privateNamespace)
	fromJid := iq.FromJID()
	toJid := iq.ToJID()
	validTo := toJid.IsServer() || toJid.Node() == fromJid.Node()
	if !validTo {
		_ = x.router.Route(ctx, iq.ForbiddenError())
		return
	}
	if iq.IsGet() {
		x.getPrivate(ctx, iq, q)
	} else if iq.IsSet() {
		x.setPrivate(ctx, iq, q)
	} else {
		_ = x.router.Route(ctx, iq.BadRequestError())
		return
	}
}

func (x *Private) getPrivate(ctx context.Context, iq *xmpp.IQ, q xmpp.XElement) {
	if q.Elements().Count() != 1 {
		_ = x.router.Route(ctx, iq.NotAcceptableError())
		return
	}
	privElem := q.Elements().All()[0]
	privNS := privElem.Namespace()
	isValidNS := x.isValidNamespace(privNS)

	if privElem.Elements().Count() > 0 || !isValidNS {
		_ = x.router.Route(ctx, iq.NotAcceptableError())
		return
	}
	fromJID := iq.FromJID()
	log.Infof("retrieving private element. ns: %s... (%s/%s)", privNS, fromJID.Node(), fromJID.Resource())

	privElements, err := storage.FetchPrivateXML(ctx, privNS, fromJID.Node())
	if err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
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

	_ = x.router.Route(ctx, res)
}

func (x *Private) setPrivate(ctx context.Context, iq *xmpp.IQ, q xmpp.XElement) {
	nsElements := map[string][]xmpp.XElement{}

	for _, privElement := range q.Elements().All() {
		ns := privElement.Namespace()
		if len(ns) == 0 {
			_ = x.router.Route(ctx, iq.BadRequestError())
			return
		}
		if !x.isValidNamespace(privElement.Namespace()) {
			_ = x.router.Route(ctx, iq.NotAcceptableError())
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
	fromJID := iq.FromJID()
	for ns, elements := range nsElements {
		log.Infof("saving private element. ns: %s... (%s/%s)", ns, fromJID.Node(), fromJID.Resource())

		if err := storage.InsertOrUpdatePrivateXML(ctx, elements, ns, fromJID.Node()); err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
	}
	_ = x.router.Route(ctx, iq.ResultIQ())
}

func (x *Private) isValidNamespace(ns string) bool {
	return !strings.HasPrefix(ns, "jabber:") && !strings.HasPrefix(ns, "http://jabber.org/") && ns != "vcard-temp"
}
