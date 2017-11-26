/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"errors"

	"github.com/ortuman/jackal/xml"
)

var IncorrectEncodingErr = errors.New("incorrect encoding")
var InvalidFormatErr = errors.New("invalid format")
var NotAuthorizedErr = errors.New("not authorized")

const saslNamespace = "urn:ietf:params:xml:ns:xmpp-sasl"

type Authenticator interface {
	Mechanism() string

	Username() string
	Authenticated() bool

	UsesChannelBinding() bool

	ProcessElement(*xml.Element) error
	Reset()
}
