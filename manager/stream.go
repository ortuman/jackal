/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package manager

import "github.com/ortuman/jackal/xml"

type C2SStream interface {
	ID() string

	Username() string
	Domain() string
	Resource() string

	JID() *xml.JID

	Secured() bool
	Authenticated() bool
	Compressed() bool

	Priority() int8

	Active() bool
	Available() bool

	RequestedRoster() bool

	SendElement(element xml.Serializable)
	Disconnect(err error)
}
