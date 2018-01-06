/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import "github.com/ortuman/jackal/xml"

type Stream interface {
	ID() string

	Username() string
	Domain() string
	Resource() string

	JID() *xml.JID

	Secured() bool
	Authenticated() bool
	Compressed() bool

	Active() bool
	Available() bool

	RequestedRoster() bool
	Priority() int8

	SendElement(element xml.Serializable)
	Disconnect(err error)
}
