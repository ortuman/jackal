/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0199

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/stream/errors"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const pingNamespace = "urn:xmpp:ping"

// Config represents XMPP Ping module (XEP-0199) configuration.
type Config struct {
	Send         bool `yaml:"send"`
	SendInterval int  `yaml:"send_interval"`
}

// Ping represents a ping server stream module.
type Ping struct {
	cfg *Config
	stm c2s.Stream

	pingTm *time.Timer
	pongCh chan struct{}

	pingMu sync.RWMutex // guards 'pingID'
	pingId string

	waitingPing uint32
	pingOnce    sync.Once
}

// New returns an ping IQ handler module.
func New(config *Config, stm c2s.Stream) *Ping {
	return &Ping{
		cfg:    config,
		stm:    stm,
		pongCh: make(chan struct{}, 1),
	}
}

// AssociatedNamespaces returns namespaces associated
// with ping module.
func (x *Ping) AssociatedNamespaces() []string {
	return []string{pingNamespace}
}

// MatchesIQ returns whether or not an IQ should be
// processed by the ping module.
func (x *Ping) MatchesIQ(iq *xml.IQ) bool {
	return x.isPongIQ(iq) || iq.Elements().ChildNamespace("ping", pingNamespace) != nil
}

// ProcessIQ processes a ping IQ taking according actions
// over the associated stream.
func (x *Ping) ProcessIQ(iq *xml.IQ) {
	if x.isPongIQ(iq) {
		x.handlePongIQ(iq)
		return
	}
	toJid := iq.ToJID()
	if !toJid.IsServer() && toJid.Node() != x.stm.Username() {
		x.stm.SendElement(iq.ForbiddenError())
		return
	}
	p := iq.Elements().ChildNamespace("ping", pingNamespace)
	if p == nil || p.Elements().Count() > 0 {
		x.stm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("received ping... id: %s", iq.ID())
	if iq.IsGet() {
		log.Infof("sent pong... id: %s", iq.ID())
		x.stm.SendElement(iq.ResultIQ())
	} else {
		x.stm.SendElement(iq.BadRequestError())
	}
}

// StartPinging starts pinging peer every 'send interval' period.
func (x *Ping) StartPinging() {
	if x.cfg.Send {
		x.pingOnce.Do(func() {
			x.pingTm = time.AfterFunc(time.Second*time.Duration(x.cfg.SendInterval), x.sendPing)
		})
	}
}

// ResetDeadline resets send ping deadline.
func (x *Ping) ResetDeadline() {
	if x.cfg.Send && atomic.LoadUint32(&x.waitingPing) == 1 {
		x.pingTm.Reset(time.Second * time.Duration(x.cfg.SendInterval))
		return
	}
}

func (x *Ping) isPongIQ(iq *xml.IQ) bool {
	x.pingMu.RLock()
	defer x.pingMu.RUnlock()
	return x.pingId == iq.ID() && (iq.IsResult() || iq.Type() == xml.ErrorType)
}

func (x *Ping) sendPing() {
	atomic.StoreUint32(&x.waitingPing, 0)

	x.pingMu.Lock()
	x.pingId = uuid.New()
	pingId := x.pingId
	x.pingMu.Unlock()

	iq := xml.NewIQType(pingId, xml.GetType)
	iq.SetTo(x.stm.JID().String())
	iq.AppendElement(xml.NewElementNamespace("ping", pingNamespace))

	x.stm.SendElement(iq)

	log.Infof("sent ping... id: %s", pingId)

	x.waitForPong()
}

func (x *Ping) waitForPong() {
	t := time.NewTimer(time.Second * time.Duration(x.cfg.SendInterval))
	select {
	case <-x.pongCh:
		return
	case <-t.C:
		x.stm.Disconnect(streamerror.ErrConnectionTimeout)
	}
}

func (x *Ping) handlePongIQ(iq *xml.IQ) {
	log.Infof("received pong... id: %s", iq.ID())

	x.pingMu.Lock()
	x.pingId = ""
	x.pingMu.Unlock()

	x.pongCh <- struct{}{}
	x.pingTm.Reset(time.Second * time.Duration(x.cfg.SendInterval))
	atomic.StoreUint32(&x.waitingPing, 1)
}
