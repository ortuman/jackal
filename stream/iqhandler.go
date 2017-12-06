/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import "github.com/ortuman/jackal/xml"

type IQHandler interface {
	MatchesIQ(*xml.IQ) bool
	ProcessIQ(*xml.IQ)
}
