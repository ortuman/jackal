/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import "github.com/ortuman/jackal/xml"

const saslNamespace = "urn:ietf:params:xml:ns:xmpp-sasl"

type Authenticator interface {
	Mechanism() string
	Username() string
	Authenticated() bool
	UsesChannelBinding() bool

	ProcessElement(xml.XElement) error
	Reset()
}

type SASLError struct {
	reason string
}

func newSASLError(reason string) error {
	return &SASLError{reason}
}

func (se *SASLError) Element() xml.XElement {
	return xml.NewElementName(se.reason)
}

func (se *SASLError) Error() string {
	return se.reason
}

var (
	ErrSASLIncorrectEncoding    = newSASLError("incorrect-encoding")
	ErrSASLMalformedRequest     = newSASLError("malformed-request")
	ErrSASLNotAuthorized        = newSASLError("not-authorized")
	ErrSASLTemporaryAuthFailure = newSASLError("temporary-auth-failure")
)
