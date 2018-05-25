/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/xml"
)

// Module represents a generic XMPP module.
type Module interface {
	// RegisterDisco registers disco entity features and items
	// associated to the module.
	RegisterDisco(discoInfo *xep0030.DiscoInfo)
}

// IQHandler represents an IQ handler module.
type IQHandler interface {
	Module

	// MatchesIQ returns whether or not an IQ should be
	// processed by the module.
	MatchesIQ(iq *xml.IQ) bool

	// ProcessIQ processes a module IQ taking according actions
	// over the associated stream.
	ProcessIQ(iq *xml.IQ)
}
