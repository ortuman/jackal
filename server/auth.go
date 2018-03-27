/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import "github.com/ortuman/jackal/xml"

const saslNamespace = "urn:ietf:params:xml:ns:xmpp-sasl"

type authenticator interface {
	Mechanism() string
	Username() string
	Authenticated() bool
	UsesChannelBinding() bool

	ProcessElement(xml.ElementNode) error
	Reset()
}

type saslError interface {
	Element() xml.ElementNode
}

type saslErrorString struct {
	reason string
}

func newSASLError(reason string) error {
	return &saslErrorString{reason}
}

func (se *saslErrorString) Element() xml.ElementNode {
	return xml.NewElementName(se.reason)
}

func (se *saslErrorString) Error() string {
	return se.reason
}

var (
	errSASLIncorrectEncoding    = newSASLError("incorrect-encoding")
	errSASLMalformedRequest     = newSASLError("malformed-request")
	errSASLNotAuthorized        = newSASLError("not-authorized")
	errSASLTemporaryAuthFailure = newSASLError("temporary-auth-failure")
)
