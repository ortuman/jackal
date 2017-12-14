/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import "github.com/ortuman/jackal/xml"

type Stream interface {
	Username() string
	Domain() string
	Resource() string

	MyJID() *xml.JID

	Authenticated() bool

	SendElement(xml.Serializable)
	SendElements([]xml.Serializable)
}

type IQHandler interface {
	AssociatedNamespaces() []string

	MatchesIQ(*xml.IQ) bool
	ProcessIQ(*xml.IQ)
}
