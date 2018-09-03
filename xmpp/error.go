/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"strconv"
)

// StanzaError represents a stanza "error" element.
type StanzaError struct {
	code      int
	errorType string
	reason    string
}

func newStanzaError(code int, errorType string, reason string) *StanzaError {
	return &StanzaError{
		code:      code,
		errorType: errorType,
		reason:    reason,
	}
}

// Error satisfies error interface.
func (se *StanzaError) Error() string {
	return se.reason
}

// Element returns StanzaError equivalent XML element.
func (se *StanzaError) Element() *Element {
	err := &Element{}
	err.SetName("error")
	err.SetAttribute("code", strconv.Itoa(se.code))
	err.SetAttribute("type", se.errorType)
	err.AppendElement(NewElementNamespace(se.reason, "urn:ietf:params:xml:ns:xmpp-stanzas"))
	return err
}

const (
	authErrorType   = "auth"
	cancelErrorType = "cancel"
	modifyErrorType = "modify"
	waitErrorType   = "wait"
)

const (
	badRequestErrorReason            = "bad-request"
	conflictErrorReason              = "conflict"
	featureNotImplementedErrorReason = "feature-not-implemented"
	forbiddenErrorReason             = "forbidden"
	goneErrorReason                  = "gone"
	internalServerErrorErrorReason   = "internal-server-error"
	itemNotFoundErrorReason          = "item-not-found"
	jidMalformedErrorReason          = "jid-malformed"
	notAcceptableErrorReason         = "not-acceptable"
	notAllowedErrorReason            = "not-allowed"
	notAuthroizedErrorReason         = "not-authorized"
	paymentRequiredErrorReason       = "payment-required"
	recipientUnavailableErrorReason  = "recipient-unavailable"
	redirectErrorReason              = "redirect"
	registrationRequiredErrorReason  = "registration-required"
	remoteServerNotFoundErrorReason  = "remote-server-not-found"
	remoteServerTimeoutErrorReason   = "remote-server-timeout"
	resourceConstraintErrorReason    = "resource-constraint"
	serviceUnavailableErrorReason    = "service-unavailable"
	subscriptionRequiredErrorReason  = "subscription-required"
	undefinedConditionErrorReason    = "undefined-condition"
	unexpectedConditionErrorReason   = "unexpected-condition"
)

var (
	// ErrBadRequest is returned by the stream when the  sender
	// has sent XML that is malformed or that cannot be processed.
	ErrBadRequest = newStanzaError(400, modifyErrorType, badRequestErrorReason)

	// ErrConflict is returned by the stream when access cannot be
	// granted because an existing resource or session exists with
	// the same name or address.
	ErrConflict = newStanzaError(409, cancelErrorType, conflictErrorReason)

	// ErrFeatureNotImplemented is returned by the stream when the feature
	// requested is not implemented by the server and therefore cannot be processed.
	ErrFeatureNotImplemented = newStanzaError(501, cancelErrorType, featureNotImplementedErrorReason)

	// ErrForbidden is returned by the stream when the requesting
	// entity does not possess the required permissions to perform the action.
	ErrForbidden = newStanzaError(403, authErrorType, forbiddenErrorReason)

	// ErrGone is returned by the stream when the recipient or server
	// can no longer be contacted at this address.
	ErrGone = newStanzaError(302, modifyErrorType, goneErrorReason)

	// ErrInternalServerError is returned by the stream when the server
	// could not process the stanza because of a misconfiguration
	// or an otherwise-undefined internal server error.
	ErrInternalServerError = newStanzaError(500, waitErrorType, internalServerErrorErrorReason)

	// ErrItemNotFound is returned by the stream when the addressed
	// JID or item requested cannot be found.
	ErrItemNotFound = newStanzaError(404, cancelErrorType, itemNotFoundErrorReason)

	// ErrJidMalformed is returned by the stream when the sending entity
	// has provided or communicated an XMPP address or aspect thereof that
	// does not adhere to the syntax defined in https://xmpp.org/rfcs/rfc3920.html#addressing.
	ErrJidMalformed = newStanzaError(400, modifyErrorType, jidMalformedErrorReason)

	// ErrNotAcceptable is returned by the stream when the server
	// understands the request but is refusing to process it because
	// it does not meet the defined criteria.
	ErrNotAcceptable = newStanzaError(406, modifyErrorType, notAcceptableErrorReason)

	// ErrNotAllowed is returned by the stream when the recipient
	// or server does not allow any entity to perform the action.
	ErrNotAllowed = newStanzaError(405, cancelErrorType, notAllowedErrorReason)

	// ErrNotAuthorized is returned by the stream when the sender
	// must provide proper credentials before being allowed to perform the action,
	// or has provided improper credentials.
	ErrNotAuthorized = newStanzaError(405, authErrorType, notAuthroizedErrorReason)

	// ErrPaymentRequired is returned by the stream when the requesting entity
	// is not authorized to access the requested service because payment is required.
	ErrPaymentRequired = newStanzaError(402, authErrorType, paymentRequiredErrorReason)

	// ErrRecipientUnavailable is returned by the stream when the intended
	// recipient is temporarily unavailable.
	ErrRecipientUnavailable = newStanzaError(404, waitErrorType, recipientUnavailableErrorReason)

	// ErrRedirect is returned by the stream when the recipient or server
	// is redirecting requests for this information to another entity, usually temporarily.
	ErrRedirect = newStanzaError(302, modifyErrorType, redirectErrorReason)

	// ErrRegistrationRequired is returned by the stream when the requesting entity
	// is not authorized to access the requested service because registration is required.
	ErrRegistrationRequired = newStanzaError(407, authErrorType, registrationRequiredErrorReason)

	// ErrRemoteServerNotFound is returned by the stream when a remote server
	// or service specified as part or all of the JID of the intended recipient does not exist.
	ErrRemoteServerNotFound = newStanzaError(404, cancelErrorType, remoteServerNotFoundErrorReason)

	// ErrRemoteServerTimeout is returned by the stream when a remote server
	// or service specified as part or all of the JID of the intended recipient
	// could not be contacted within a reasonable amount of time.
	ErrRemoteServerTimeout = newStanzaError(504, waitErrorType, remoteServerTimeoutErrorReason)

	// ErrResourceConstraint is returned by the stream when the server or recipient
	// lacks the system resources necessary to service the request.
	ErrResourceConstraint = newStanzaError(500, waitErrorType, resourceConstraintErrorReason)

	// ErrServiceUnavailable is returned by the stream when the server or recipient
	// does not currently provide the requested service.
	ErrServiceUnavailable = newStanzaError(503, cancelErrorType, serviceUnavailableErrorReason)

	// ErrSubscriptionRequired is returned by the stream when the requesting entity
	// is not authorized to access the requested service because a subscription is required.
	ErrSubscriptionRequired = newStanzaError(407, authErrorType, subscriptionRequiredErrorReason)

	// ErrUndefinedCondition is returned by the stream when the error condition
	// is not one of those defined by the other conditions in this list.
	ErrUndefinedCondition = newStanzaError(500, waitErrorType, undefinedConditionErrorReason)

	// ErrUnexpectedCondition is returned by the stream when the recipient or server
	// understood the request but was not expecting it at this time.
	ErrUnexpectedCondition = newStanzaError(400, waitErrorType, unexpectedConditionErrorReason)
)

// BadRequestError returns an error copy of the element
// attaching 'bad-request' error sub element.
func (stanza *stanzaElement) BadRequestError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrBadRequest, nil)
}

// ConflictError returns an error copy of the element
// attaching 'conflict' error sub element.
func (stanza *stanzaElement) ConflictError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrConflict, nil)
}

// FeatureNotImplementedError returns an error copy of the element
// attaching 'feature-not-implemented' error sub element.
func (stanza *stanzaElement) FeatureNotImplementedError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrFeatureNotImplemented, nil)
}

// ForbiddenError returns an error copy of the element
// attaching 'forbidden' error sub element.
func (stanza *stanzaElement) ForbiddenError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrForbidden, nil)
}

// GoneError returns an error copy of the element
// attaching 'gone' error sub element.
func (stanza *stanzaElement) GoneError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrGone, nil)
}

// InternalServerError returns an error copy of the element
// attaching 'internal-server-error' error sub element.
func (stanza *stanzaElement) InternalServerError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrInternalServerError, nil)
}

// ItemNotFoundError returns an error copy of the element
// attaching 'item-not-found' error sub element.
func (stanza *stanzaElement) ItemNotFoundError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrItemNotFound, nil)
}

// JidMalformedError returns an error copy of the element
// attaching 'jid-malformed' error sub element.
func (stanza *stanzaElement) JidMalformedError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrJidMalformed, nil)
}

// NotAcceptableError returns an error copy of the element
// attaching 'not-acceptable' error sub element.
func (stanza *stanzaElement) NotAcceptableError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrNotAcceptable, nil)
}

// NotAllowedError returns an error copy of the element
// attaching 'not-allowed' error sub element.
func (stanza *stanzaElement) NotAllowedError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrNotAllowed, nil)
}

// NotAuthorizedError returns an error copy of the element
// attaching 'not-authorized' error sub element.
func (stanza *stanzaElement) NotAuthorizedError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrNotAuthorized, nil)
}

// PaymentRequiredError returns an error copy of the element
// attaching 'payment-required' error sub element.
func (stanza *stanzaElement) PaymentRequiredError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrPaymentRequired, nil)
}

// RecipientUnavailableError returns an error copy of the element
// attaching 'recipient-unavailable' error sub element.
func (stanza *stanzaElement) RecipientUnavailableError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrRecipientUnavailable, nil)
}

// RedirectError returns an error copy of the element
// attaching 'redirect' error sub element.
func (stanza *stanzaElement) RedirectError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrRedirect, nil)
}

// RegistrationRequiredError returns an error copy of the element
// attaching 'registration-required' error sub element.
func (stanza *stanzaElement) RegistrationRequiredError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrRegistrationRequired, nil)
}

// RemoteServerNotFoundError returns an error copy of the element
// attaching 'remote-server-not-found' error sub element.
func (stanza *stanzaElement) RemoteServerNotFoundError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrRemoteServerNotFound, nil)
}

// RemoteServerTimeoutError returns an error copy of the element
// attaching 'remote-server-timeout' error sub element.
func (stanza *stanzaElement) RemoteServerTimeoutError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrRemoteServerTimeout, nil)
}

// ResourceConstraintError returns an error copy of the element
// attaching 'resource-constraint' error sub element.
func (stanza *stanzaElement) ResourceConstraintError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrResourceConstraint, nil)
}

// ServiceUnavailableError returns an error copy of the element
// attaching 'service-unavailable' error sub element.
func (stanza *stanzaElement) ServiceUnavailableError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrServiceUnavailable, nil)
}

// SubscriptionRequiredError returns an error copy of the element
// attaching 'subscription-required' error sub element.
func (stanza *stanzaElement) SubscriptionRequiredError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrSubscriptionRequired, nil)
}

// UndefinedConditionError returns an error copy of the element
// attaching 'undefined-condition' error sub element.
func (stanza *stanzaElement) UndefinedConditionError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrUndefinedCondition, nil)
}

// UnexpectedConditionError returns an error copy of the element
// attaching 'unexpected-condition' error sub element.
func (stanza *stanzaElement) UnexpectedConditionError() Stanza {
	return NewErrorStanzaFromStanza(stanza, ErrUnexpectedCondition, nil)
}
