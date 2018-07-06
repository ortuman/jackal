/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import "github.com/ortuman/jackal/xml"

const saslNamespace = "urn:ietf:params:xml:ns:xmpp-sasl"

// Authenticator defines a generic authenticator state machine.
type Authenticator interface {

	// Mechanism returns authenticator mechanism name.
	Mechanism() string

	// Username returns authenticated username in case
	// authentication process has been completed.
	Username() string

	// Authenticated returns whether or not user has been authenticated.
	Authenticated() bool

	// UsesChannelBinding returns whether or not this authenticator
	// requires channel binding bytes.
	UsesChannelBinding() bool

	// ProcessElement process an incoming authenticator element.
	ProcessElement(xml.XElement) error

	// Reset resets authenticator internal state.
	Reset()
}

// SASLError represents specific SASL error type.
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
