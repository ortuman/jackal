/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/stream/errors"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const pingNamespace = "urn:xmpp:ping"

type XEPPing struct {
	cfg  *config.ModPing
	strm stream.C2SStream

	pingTm *time.Timer
	pongCh chan struct{}

	pingMu sync.RWMutex // guards 'pingID'
	pingId string

	waitingPing uint32
	pingOnce    sync.Once
}

func NewXEPPing(config *config.ModPing, strm stream.C2SStream) *XEPPing {
	return &XEPPing{
		cfg:    config,
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
	log.Infof("received ping... id: %s", iq.ID())
	if iq.IsGet() {
		log.Infof("sent pong... id: %s", iq.ID())
		x.strm.SendElement(iq.ResultIQ())
	} else {
		x.strm.SendElement(iq.BadRequestError())
	}
}

func (x *XEPPing) StartPinging() {
	if !x.cfg.Send {
		return
	}
	x.pingOnce.Do(func() {
		x.pingTm = time.AfterFunc(time.Second*time.Duration(x.cfg.SendInterval), x.sendPing)
	})
}

func (x *XEPPing) ResetDeadline() {
	if !x.cfg.Send {
		return
	}
	if atomic.LoadUint32(&x.waitingPing) == 1 {
		x.pingTm.Reset(time.Second * time.Duration(x.cfg.SendInterval))
	}
}

func (x *XEPPing) isPongIQ(iq *xml.IQ) bool {
	x.pingMu.RLock()
	defer x.pingMu.RUnlock()
	return x.pingId == iq.ID() && (iq.IsResult() || iq.IsError())
}

func (x *XEPPing) sendPing() {
	atomic.StoreUint32(&x.waitingPing, 0)

	x.pingMu.Lock()
	x.pingId = uuid.New()
	pingId := x.pingId
	x.pingMu.Unlock()

	iq := xml.NewIQType(pingId, xml.GetType)
	iq.SetTo(x.strm.JID().String())
	iq.AppendElement(xml.NewElementNamespace("ping", pingNamespace))

	x.strm.SendElement(iq)

	log.Infof("sent ping... id: %s", pingId)

	x.waitForPong()
}

func (x *XEPPing) waitForPong() {
	t := time.NewTimer(time.Second * time.Duration(x.cfg.SendInterval))
	select {
	case <-x.pongCh:
		return
	case <-t.C:
		x.strm.Disconnect(streamerror.ErrConnectionTimeout)
	}
}

func (x *XEPPing) handlePongIQ(iq *xml.IQ) {
	log.Infof("received pong... id: %s", iq.ID())

	x.pingMu.Lock()
	x.pingId = ""
	x.pingMu.Unlock()

	x.pongCh <- struct{}{}
	x.pingTm.Reset(time.Second * time.Duration(x.cfg.SendInterval))
	atomic.StoreUint32(&x.waitingPing, 1)
}
