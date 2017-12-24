/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const pingNamespace = "urn:xmpp:ping"

type XEPPing struct {
	cfg  *config.ModPing
	strm Stream

	recv   uint32
	pongCh chan struct{}

	pingMu sync.RWMutex // guards 'pingID'
	pingID string
}

func NewXEPPing(cfg *config.ModPing, strm Stream) *XEPPing {
	return &XEPPing{
		cfg:    cfg,
		strm:   strm,
		pongCh: make(chan struct{}, 1),
	}
}

func (x *XEPPing) AssociatedNamespaces() []string {
	return []string{pingNamespace}
}

func (x *XEPPing) MatchesIQ(iq *xml.IQ) bool {
	return x.isPongIQ(iq) || iq.FindElementNamespace("ping", pingNamespace) != nil
}

func (x *XEPPing) ProcessIQ(iq *xml.IQ) {
	if x.isPongIQ(iq) {
		x.handlePongIQ(iq)
		return
	}
	toJid := iq.ToJID()
	if toJid.IsBare() && toJid.Node() != x.strm.Username() {
		x.strm.SendElement(iq.ForbiddenError())
		return
	}
	p := iq.FindElementNamespace("ping", pingNamespace)
	if p.ElementsCount() > 0 {
		x.strm.SendElement(iq.BadRequestError())
		return
	}
	if iq.IsGet() {
		x.strm.SendElement(iq.ResultIQ())
	} else {
		x.strm.SendElement(iq.BadRequestError())
	}
}

func (x *XEPPing) StartPinging() {
	if !x.cfg.Send {
		return
	}
	go x.startPinging()
}

func (x *XEPPing) NotifyReceive() {
	if !x.cfg.Send {
		return
	}
	atomic.CompareAndSwapUint32(&x.recv, 0, 1)
}

func (x *XEPPing) isPongIQ(iq *xml.IQ) bool {
	x.pingMu.RLock()
	defer x.pingMu.RUnlock()
	return x.pingID == iq.ID() && (iq.IsResult() || iq.IsError())
}

func (x *XEPPing) startPinging() {
	t := time.NewTicker(time.Second * time.Duration(x.cfg.SendInterval))
	defer t.Stop()
	for {
		<-t.C
		if atomic.CompareAndSwapUint32(&x.recv, 1, 0) {
			continue
		} else {
			pingID := uuid.New()
			x.pingMu.Lock()
			x.pingID = pingID
			x.pingMu.Unlock()

			iq := xml.NewMutableIQType(pingID, xml.GetType)
			iq.AppendElement(xml.NewElementNamespace("ping", pingNamespace))
			x.strm.SendElement(iq)
			x.waitForPong()
			return
		}
	}
}

func (x *XEPPing) waitForPong() {
	t := time.NewTimer(time.Duration(x.cfg.SendInterval) / 3)
	select {
	case <-x.pongCh:
		return
	case <-t.C:
		x.strm.Disconnect(errors.ErrConnectionTimeout)
	}
}

func (x *XEPPing) handlePongIQ(iq *xml.IQ) {
	x.pingMu.Lock()
	x.pingID = ""
	x.pingMu.Unlock()

	x.pongCh <- struct{}{}
	go x.startPinging()
}
