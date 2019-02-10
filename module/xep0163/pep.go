/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 2048

var discoInfoFeatures = []string{
	"http://jabber.org/protocol/pubsub#access-presence",
	"http://jabber.org/protocol/pubsub#auto-create",
	"http://jabber.org/protocol/pubsub#auto-subscribe",
	"http://jabber.org/protocol/pubsub#config-node",
	"http://jabber.org/protocol/pubsub#create-and-configure",
	"http://jabber.org/protocol/pubsub#create-nodes",
	"http://jabber.org/protocol/pubsub#filtered-notifications",
	"http://jabber.org/protocol/pubsub#persistent-items",
	"http://jabber.org/protocol/pubsub#publish",
	"http://jabber.org/protocol/pubsub#retrieve-items",
	"http://jabber.org/protocol/pubsub#subscribe",
}

type Pep struct {
	router     *router.Router
	actorCh    chan func()
	shutdownCh chan chan error
}

func New(disco *xep0030.DiscoInfo, router *router.Router) *Pep {
	p := &Pep{
		router:     router,
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan error),
	}
	go p.loop()

	// register account features
	if disco != nil {
		for _, feature := range discoInfoFeatures {
			disco.RegisterAccountFeature(feature)
		}
	}
	return p
}

// MatchesIQ returns whether or not an IQ should be
// processed by the version module.
func (x *Pep) MatchesIQ(iq *xmpp.IQ) bool {
	return false
}

// ProcessIQ processes a version IQ taking according actions
// over the associated stream.
func (x *Pep) ProcessIQ(iq *xmpp.IQ) {
	x.actorCh <- func() {
		x.processIQ(iq)
	}
}

// Shutdown shuts down version module.
func (x *Pep) Shutdown() error {
	c := make(chan error)
	x.shutdownCh <- c
	return <-c
}

// runs on it's own goroutine
func (x *Pep) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case c := <-x.shutdownCh:
			c <- nil
			return
		}
	}
}

func (x *Pep) processIQ(iq *xmpp.IQ) {
}
