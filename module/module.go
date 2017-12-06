/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import "github.com/ortuman/jackal/xml"

type IQHandler interface {
	MatchesIQ(*xml.IQ) bool
	ProcessIQ(*xml.IQ)
}

type Stream interface {
	Username() string
	Domain() string
	Resource() string

	Authenticated() bool

	SendElement(xml.Serializable)
	SendElements([]xml.Serializable)
}
