/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"errors"

	"github.com/ortuman/jackal/xml"
)

var errIncorrectEncoding = errors.New("incorrect encoding")
var errInvalidFormat = errors.New("invalid format")
var errNotAuthorized = errors.New("not authorized")

const saslNamespace = "urn:ietf:params:xml:ns:xmpp-sasl"

type authenticator interface {
	Mechanism() string

	Username() string
	Authenticated() bool

	UsesChannelBinding() bool

	ProcessElement(*xml.Element) error
	Reset()
}
