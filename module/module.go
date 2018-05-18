/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"github.com/ortuman/jackal/xml"
)

// IQHandler represents an IQ handler module.
type IQHandler interface {
	// MatchesIQ returns whether or not an IQ should be
	// processed by this module.
	MatchesIQ(iq *xml.IQ) bool

	// ProcessIQ processes a module IQ taking according actions
	// over the associated stream.
	ProcessIQ(iq *xml.IQ)
}
