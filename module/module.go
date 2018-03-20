/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/xml"
)

const moduleMailboxSize = 16

// Module represents an XMPP module.
type Module interface {
	// AssociatedNamespaces returns namespaces associated
	// with this module.
	AssociatedNamespaces() []string

	// Done signals stream termination.
	Done()
}

// IQHandler represents an IQ handler module.
type IQHandler interface {
	Module

	// MatchesIQ returns whether or not an IQ should be
	// processed by this module.
	MatchesIQ(iq *xml.IQ) bool

	// ProcessIQ processes a module IQ taking according actions
	// over the associated stream.
	ProcessIQ(iq *xml.IQ)
}
