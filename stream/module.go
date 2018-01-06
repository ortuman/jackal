/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"github.com/ortuman/jackal/xml"
)

type Module interface {
	AssociatedNamespaces() []string
}

type IQHandler interface {
	Module
	MatchesIQ(*xml.IQ) bool
	ProcessIQ(*xml.IQ)
}
