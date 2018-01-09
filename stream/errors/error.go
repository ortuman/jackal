/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package streamerror

import (
	"github.com/ortuman/jackal/xml"
)

// Error represents a "stream:error" element.
type Error struct {
	reason string
}

var (
	// ErrInvalidXML represents 'invalid-xml' stream error.
	ErrInvalidXML = newStreamError("invalid-xml")

	// ErrInvalidNamespace represents 'invalid-namespace' stream error.
	ErrInvalidNamespace = newStreamError("invalid-namespace")

	// ErrHostUnknown represents 'host-unknown' stream error.
	ErrHostUnknown = newStreamError("host-unknown")

	// ErrInvalidFrom represents 'invalid-from' stream error.
	ErrInvalidFrom = newStreamError("invalid-from")

	// ErrConnectionTimeout represents 'connection-timeout' stream error.
	ErrConnectionTimeout = newStreamError("connection-timeout")

	// ErrUnsupportedStanzaType represents 'unsupported-stanza-type' stream error.
	ErrUnsupportedStanzaType = newStreamError("unsupported-stanza-type")

	// ErrUnsupportedVersion represents 'unsupported-version' stream error.
	ErrUnsupportedVersion = newStreamError("unsupported-version")

	// ErrNotAuthorized represents 'not-authorized' stream error.
	ErrNotAuthorized = newStreamError("not-authorized")

	// ErrInternalServerError represents 'internal-server-error' stream error.
	ErrInternalServerError = newStreamError("internal-server-error")
)

func newStreamError(reason string) *Error {
	return &Error{reason: reason}
}

func (se *Error) Element() xml.Element {
	ret := xml.NewElementName("stream:error")
	reason := xml.NewElementNamespace(se.reason, "urn:ietf:params:xml:ns:xmpp-streams")
	ret.AppendElement(reason)
	return ret
}

func (se *Error) Error() string {
	return se.reason
}
