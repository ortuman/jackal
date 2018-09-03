/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0199

import (
	"fmt"
	"time"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
)

const mailboxSize = 2048

const pingNamespace = "urn:xmpp:ping"

// Config represents XMPP Ping module (XEP-0199) configuration.
type Config struct {
	Send         bool
	SendInterval time.Duration
}

type configProxy struct {
	Send         bool `yaml:"send"`
	SendInterval int  `yaml:"send_interval"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := configProxy{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	c.Send = p.Send
	c.SendInterval = time.Second * time.Duration(p.SendInterval)
	if c.Send && c.SendInterval < time.Second {
		return fmt.Errorf("xep0199.Config: send interval must be 1 or higher")
	}
	return nil
}

type ping struct {
	identifier string
	timer      *time.Timer
	stm        stream.C2S
}

// Ping represents a ping server stream module.
type Ping struct {
	cfg         *Config
	pings       map[string]*ping
	activePings map[string]*ping
	actorCh     chan func()
	shutdownCh  <-chan struct{}
}

// New returns an ping IQ handler module.
func New(config *Config, disco *xep0030.DiscoInfo, shutdownCh <-chan struct{}) *Ping {
	p := &Ping{
		cfg:         config,
		pings:       make(map[string]*ping),
		activePings: make(map[string]*ping),
		actorCh:     make(chan func(), mailboxSize),
		shutdownCh:  shutdownCh,
	}
	go p.loop()
	if disco != nil {
		disco.RegisterServerFeature(pingNamespace)
		disco.RegisterAccountFeature(pingNamespace)
	}
	return p
}

// MatchesIQ returns whether or not an IQ should be
// processed by the ping module.
func (x *Ping) MatchesIQ(iq *xmpp.IQ) bool {
	return x.isPongIQ(iq) || iq.Elements().ChildNamespace("ping", pingNamespace) != nil
}

// ProcessIQ processes a ping IQ taking according actions
// over the associated stream.
func (x *Ping) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	x.actorCh <- func() { x.processIQ(iq, stm) }
}

// SchedulePing schedules a new ping in a 'send interval' period,
// cancelling previous scheduled ping.
func (x *Ping) SchedulePing(stm stream.C2S) {
	x.actorCh <- func() { x.schedulePing(stm) }
}

// CancelPing cancels a previous scheduled ping.
func (x *Ping) CancelPing(stm stream.C2S) {
	x.actorCh <- func() { x.cancelPing(stm) }
}

// runs on it's own goroutine
func (x *Ping) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case <-x.shutdownCh:
			for _, pi := range x.pings {
				pi.timer.Stop()
			}
			return
		}
	}
}

func (x *Ping) processIQ(iq *xmpp.IQ, stm stream.C2S) {
	if x.isPongIQ(iq) {
		x.handlePongIQ(iq, stm)
		return
	}
	toJid := iq.ToJID()
	if !toJid.IsServer() && toJid.Node() != stm.Username() {
		stm.SendElement(iq.ForbiddenError())
		return
	}
	p := iq.Elements().ChildNamespace("ping", pingNamespace)
	if p == nil || p.Elements().Count() > 0 {
		stm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("received ping... id: %s", iq.ID())
	if iq.IsGet() {
		log.Infof("sent pong... id: %s", iq.ID())
		stm.SendElement(iq.ResultIQ())
	} else {
		stm.SendElement(iq.BadRequestError())
	}
}

func (x *Ping) schedulePing(stm stream.C2S) {
	if !x.cfg.Send || !stm.JID().IsFull() {
		return
	}
	userJID := stm.JID().String()

	if pi := x.pings[userJID]; pi != nil {
		if _, ok := x.activePings[pi.identifier]; ok {
			// waiting for pong
			return
		}
		// cancel previous ping
		pi.timer.Stop()
	}
	x.schedulePingTimer(stm)
}

func (x *Ping) cancelPing(stm stream.C2S) {
	if !x.cfg.Send || !stm.JID().IsFull() {
		return
	}
	userJID := stm.JID().String()

	if pi := x.pings[userJID]; pi != nil {
		pi.timer.Stop()

		delete(x.pings, userJID)
		delete(x.activePings, pi.identifier)
	}
}

func (x *Ping) schedulePingTimer(stm stream.C2S) {
	pi := &ping{
		identifier: uuid.New(),
		stm:        stm,
	}
	pi.timer = time.AfterFunc(x.cfg.SendInterval, func() {
		x.actorCh <- func() { x.sendPing(pi) }
	})
	x.pings[stm.JID().String()] = pi
}

func (x *Ping) handlePongIQ(iq *xmpp.IQ, stm stream.C2S) {
	pongID := iq.ID()
	if pi := x.activePings[pongID]; pi != nil && pi.stm == stm {
		log.Infof("received pong... id: %s", pongID)

		pi.timer.Stop()
		x.schedulePingTimer(stm)
	}
}

func (x *Ping) sendPing(pi *ping) {
	srvJID, _ := jid.New("", pi.stm.JID().Domain(), "", true)

	iq := xmpp.NewIQType(pi.identifier, xmpp.GetType)
	iq.SetFromJID(srvJID)
	iq.SetToJID(pi.stm.JID())
	iq.AppendElement(xmpp.NewElementNamespace("ping", pingNamespace))

	pi.stm.SendElement(iq)

	log.Infof("sent ping... id: %s", pi.identifier)

	pi.timer = time.AfterFunc(x.cfg.SendInterval/3, func() {
		x.actorCh <- func() { x.disconnectStream(pi) }
	})
	x.activePings[pi.identifier] = pi
}

func (x *Ping) disconnectStream(pi *ping) {
	pi.stm.Disconnect(streamerror.ErrConnectionTimeout)
}

func (x *Ping) isPongIQ(iq *xmpp.IQ) bool {
	_, ok := x.activePings[iq.ID()]
	return ok && (iq.IsResult() || iq.Type() == xmpp.ErrorType)
}
