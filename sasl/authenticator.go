/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package sasl

import "github.com/ortuman/jackal/xml"

type Authenticator interface {
	Mechanism() string

	Username() string
	Authenticated() bool

	UsesChannelBinding() bool

	ProcessElement(*xml.Element) error
	Reset()
}
