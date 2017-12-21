/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/xml"
)

const pingNamespace = "urn:xmpp:ping"

type XEPPing struct {
	cfg    *config.ModPing
	strm   Stream
	pingID string
}

func NewXEPPing(cfg *config.ModPing, strm Stream) *XEPPing {
	return &XEPPing{cfg: cfg, strm: strm}
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

func (x *XEPPing) isPongIQ(iq *xml.IQ) bool {
	return x.pingID == iq.ID() && (iq.IsResult() || iq.IsError())
}

func (x *XEPPing) handlePongIQ(iq *xml.IQ) {
}
