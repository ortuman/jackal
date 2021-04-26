// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"fmt"

	"github.com/jackal-xmpp/stravaganza/v2"
)

const saslNamespace = "urn:ietf:params:xml:ns:xmpp-sasl"

// Authenticator defines a generic authenticator state machine.
type Authenticator interface {

	// Mechanism returns authenticator mechanism name.
	Mechanism() string

	// Username returns authenticated username in case authentication process has been completed.
	Username() string

	// Authenticated returns whether or not user has been authenticated.
	Authenticated() bool

	// UsesChannelBinding returns whether or not this authenticator requires channel binding bytes.
	UsesChannelBinding() bool

	// ProcessElement process an incoming authenticator element.
	ProcessElement(context.Context, stravaganza.Element) (stravaganza.Element, *SASLError)

	// Reset resets authenticator internal state.
	Reset()
}

// SASLErrorReason defines the SASL error reason.
type SASLErrorReason uint8

const (
	// IncorrectEncoding represents a 'incorrect-encoding' authentication error.
	IncorrectEncoding SASLErrorReason = iota

	// MalformedRequest represents a 'malformed-request' authentication error.
	MalformedRequest

	// NotAuthorized represents a 'not-authorized' authentication error.
	NotAuthorized

	// TemporaryAuthFailure represents a 'temporary-auth-failure' authentication error.
	TemporaryAuthFailure
)

// String returns SASLErrorReason string representation.
func (r SASLErrorReason) String() string {
	switch r {
	case IncorrectEncoding:
		return "incorrect-encoding"
	case MalformedRequest:
		return "malformed-request"
	case NotAuthorized:
		return "not-authorized"
	case TemporaryAuthFailure:
		return "temporary-auth-failure"
	default:
		return ""
	}
}

// SASLError represents specific SASL error type.
type SASLError struct {
	Reason SASLErrorReason
	Err    error
}

func newSASLError(reason SASLErrorReason, err error) *SASLError {
	return &SASLError{Reason: reason, Err: err}
}

// Element returs sasl error XML representation.
func (se *SASLError) Element() stravaganza.Element {
	return stravaganza.NewBuilder(se.Reason.String()).Build()
}

// Error satisfies error interface.
func (se *SASLError) Error() string {
	if se.Reason != TemporaryAuthFailure && se.Err != nil {
		return fmt.Sprintf("%s: %v", se.Reason, se.Err)
	}
	return se.Reason.String()
}
