/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package auth

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

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
	ProcessElement(context.Context, xmpp.XElement) error

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

// Element returs sasl error XML representation.
func (se *SASLError) Element() xmpp.XElement {
	return xmpp.NewElementName(se.reason)
}

// Error satisfies error interface.
func (se *SASLError) Error() string {
	return se.reason
}

var (
	// ErrSASLIncorrectEncoding represents a 'incorrect-encoding' authentication error.
	ErrSASLIncorrectEncoding = newSASLError("incorrect-encoding")

	// ErrSASLMalformedRequest represents a 'malformed-request' authentication error.
	ErrSASLMalformedRequest = newSASLError("malformed-request")

	// ErrSASLNotAuthorized represents a 'not-authorized' authentication error.
	ErrSASLNotAuthorized = newSASLError("not-authorized")

	// ErrSASLTemporaryAuthFailure represents a 'temporary-auth-failure' authentication error.
	ErrSASLTemporaryAuthFailure = newSASLError("temporary-auth-failure")
)
