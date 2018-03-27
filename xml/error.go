/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"strconv"
)

// StanzaError represents a stanza "error" element.
type StanzaError struct {
	code      int
	errorType string
	reason    string
}

func newErrorElement(code int, errorType string, reason string) error {
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
func (se *StanzaError) Element() Element {
	err := &xElement{name: "error"}
	err.Attributes().setAttribute("code", strconv.Itoa(se.code))
	err.Attributes().setAttribute("type", se.errorType)
	err.appendElement(NewElementNamespace(se.reason, "urn:ietf:params:xml:ns:xmpp-stanzas"))
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
	ErrBadRequest = newErrorElement(400, modifyErrorType, badRequestErrorReason)

	// ErrConflict is returned by the stream when access cannot be
	// granted because an existing resource or session exists with
	// the same name or address.
	ErrConflict = newErrorElement(409, cancelErrorType, conflictErrorReason)

	// ErrFeatureNotImplemented is returned by the stream when the feature
	// requested is not implemented by the server and therefore cannot be processed.
	ErrFeatureNotImplemented = newErrorElement(501, cancelErrorType, featureNotImplementedErrorReason)

	// ErrForbidden is returned by the stream when the requesting
	// entity does not possess the required permissions to perform the action.
	ErrForbidden = newErrorElement(403, authErrorType, forbiddenErrorReason)

	// ErrGone is returned by the stream when the recipient or server
	// can no longer be contacted at this address.
	ErrGone = newErrorElement(302, modifyErrorType, goneErrorReason)

	// ErrInternalServerError is returned by the stream when the server
	// could not process the stanza because of a misconfiguration
	// or an otherwise-undefined internal server error.
	ErrInternalServerError = newErrorElement(500, waitErrorType, internalServerErrorErrorReason)

	// ErrItemNotFound is returned by the stream when the addressed
	// JID or item requested cannot be found.
	ErrItemNotFound = newErrorElement(404, cancelErrorType, itemNotFoundErrorReason)

	// ErrJidMalformed is returned by the stream when the sending entity
	// has provided or communicated an XMPP address or aspect thereof that
	// does not adhere to the syntax defined in https://xmpp.org/rfcs/rfc3920.html#addressing.
	ErrJidMalformed = newErrorElement(400, modifyErrorType, jidMalformedErrorReason)

	// ErrNotAcceptable is returned by the stream when the server
	// understands the request but is refusing to process it because
	// it does not meet the defined criteria.
	ErrNotAcceptable = newErrorElement(406, modifyErrorType, notAcceptableErrorReason)

	// ErrNotAllowed is returned by the stream when the recipient
	// or server does not allow any entity to perform the action.
	ErrNotAllowed = newErrorElement(405, cancelErrorType, notAllowedErrorReason)

	// ErrNotAuthorized is returned by the stream when the sender
	// must provide proper credentials before being allowed to perform the action,
	// or has provided improper credentials.
	ErrNotAuthorized = newErrorElement(405, authErrorType, notAuthroizedErrorReason)

	// ErrPaymentRequired is returned by the stream when the requesting entity
	// is not authorized to access the requested service because payment is required.
	ErrPaymentRequired = newErrorElement(402, authErrorType, paymentRequiredErrorReason)

	// ErrRecipientUnavailable is returned by the stream when the intended
	// recipient is temporarily unavailable.
	ErrRecipientUnavailable = newErrorElement(404, waitErrorType, recipientUnavailableErrorReason)

	// ErrRedirect is returned by the stream when the recipient or server
	// is redirecting requests for this information to another entity, usually temporarily.
	ErrRedirect = newErrorElement(302, modifyErrorType, redirectErrorReason)

	// ErrRegistrationRequired is returned by the stream when the requesting entity
	// is not authorized to access the requested service because registration is required.
	ErrRegistrationRequired = newErrorElement(407, authErrorType, registrationRequiredErrorReason)

	// ErrRemoteServerNotFound is returned by the stream when a remote server
	// or service specified as part or all of the JID of the intended recipient does not exist.
	ErrRemoteServerNotFound = newErrorElement(404, cancelErrorType, remoteServerNotFoundErrorReason)

	// ErrRemoteServerTimeout is returned by the stream when a remote server
	// or service specified as part or all of the JID of the intended recipient
	// could not be contacted within a reasonable amount of time.
	ErrRemoteServerTimeout = newErrorElement(504, waitErrorType, remoteServerTimeoutErrorReason)

	// ErrResourceConstraint is returned by the stream when the server or recipient
	// lacks the system resources necessary to service the request.
	ErrResourceConstraint = newErrorElement(500, waitErrorType, resourceConstraintErrorReason)

	// ErrServiceUnavailable is returned by the stream when the server or recipient
	// does not currently provide the requested service.
	ErrServiceUnavailable = newErrorElement(503, cancelErrorType, serviceUnavailableErrorReason)

	// ErrSubscriptionRequired is returned by the stream when the requesting entity
	// is not authorized to access the requested service because a subscription is required.
	ErrSubscriptionRequired = newErrorElement(407, authErrorType, subscriptionRequiredErrorReason)

	// ErrUndefinedCondition is returned by the stream when the error condition
	// is not one of those defined by the other conditions in this list.
	ErrUndefinedCondition = newErrorElement(500, waitErrorType, undefinedConditionErrorReason)

	// ErrUnexpectedCondition is returned by the stream when the recipient or server
	// understood the request but was not expecting it at this time.
	ErrUnexpectedCondition = newErrorElement(400, waitErrorType, unexpectedConditionErrorReason)
)

// ToError returns an error copy element attaching
// stanza error sub element.
func (el *xElement) ToError(stanzaError *StanzaError) Element {
	errEl := &xElement{}
	errEl.copyFrom(el)
	errEl.Attributes().setAttribute("type", "error")
	errEl.appendElement(stanzaError.Element())
	return errEl
}

// BadRequestError returns an error copy of the element
// attaching 'bad-request' error sub element.
func (el *xElement) BadRequestError() Element {
	return el.ToError(ErrBadRequest.(*StanzaError))
}

// ConflictError returns an error copy of the element
// attaching 'conflict' error sub element.
func (el *xElement) ConflictError() Element {
	return el.ToError(ErrConflict.(*StanzaError))
}

// FeatureNotImplementedError returns an error copy of the element
// attaching 'feature-not-implemented' error sub element.
func (el *xElement) FeatureNotImplementedError() Element {
	return el.ToError(ErrFeatureNotImplemented.(*StanzaError))
}

// ForbiddenError returns an error copy of the element
// attaching 'forbidden' error sub element.
func (el *xElement) ForbiddenError() Element {
	return el.ToError(ErrForbidden.(*StanzaError))
}

// GoneError returns an error copy of the element
// attaching 'gone' error sub element.
func (el *xElement) GoneError() Element {
	return el.ToError(ErrGone.(*StanzaError))
}

// InternalServerError returns an error copy of the element
// attaching 'internal-server-error' error sub element.
func (el *xElement) InternalServerError() Element {
	return el.ToError(ErrInternalServerError.(*StanzaError))
}

// ItemNotFoundError returns an error copy of the element
// attaching 'item-not-found' error sub element.
func (el *xElement) ItemNotFoundError() Element {
	return el.ToError(ErrItemNotFound.(*StanzaError))
}

// JidMalformedError returns an error copy of the element
// attaching 'jid-malformed' error sub element.
func (el *xElement) JidMalformedError() Element {
	return el.ToError(ErrJidMalformed.(*StanzaError))
}

// NotAcceptableError returns an error copy of the element
// attaching 'not-acceptable' error sub element.
func (el *xElement) NotAcceptableError() Element {
	return el.ToError(ErrNotAcceptable.(*StanzaError))
}

// NotAllowedError returns an error copy of the element
// attaching 'not-allowed' error sub element.
func (el *xElement) NotAllowedError() Element {
	return el.ToError(ErrNotAllowed.(*StanzaError))
}

// NotAuthorizedError returns an error copy of the element
// attaching 'not-authorized' error sub element.
func (el *xElement) NotAuthorizedError() Element {
	return el.ToError(ErrNotAuthorized.(*StanzaError))
}

// PaymentRequiredError returns an error copy of the element
// attaching 'payment-required' error sub element.
func (el *xElement) PaymentRequiredError() Element {
	return el.ToError(ErrPaymentRequired.(*StanzaError))
}

// RecipientUnavailableError returns an error copy of the element
// attaching 'recipient-unavailable' error sub element.
func (el *xElement) RecipientUnavailableError() Element {
	return el.ToError(ErrRecipientUnavailable.(*StanzaError))
}

// RedirectError returns an error copy of the element
// attaching 'redirect' error sub element.
func (el *xElement) RedirectError() Element {
	return el.ToError(ErrRedirect.(*StanzaError))
}

// RegistrationRequiredError returns an error copy of the element
// attaching 'registration-required' error sub element.
func (el *xElement) RegistrationRequiredError() Element {
	return el.ToError(ErrRegistrationRequired.(*StanzaError))
}

// RemoteServerNotFoundError returns an error copy of the element
// attaching 'remote-server-not-found' error sub element.
func (el *xElement) RemoteServerNotFoundError() Element {
	return el.ToError(ErrRemoteServerNotFound.(*StanzaError))
}

// RemoteServerNotFoundError returns an error copy of the element
// attaching 'remote-server-timeout' error sub element.
func (el *xElement) RemoteServerTimeoutError() Element {
	return el.ToError(ErrRemoteServerTimeout.(*StanzaError))
}

// ResourceConstraintError returns an error copy of the element
// attaching 'resource-constraint' error sub element.
func (el *xElement) ResourceConstraintError() Element {
	return el.ToError(ErrResourceConstraint.(*StanzaError))
}

// ServiceUnavailableError returns an error copy of the element
// attaching 'service-unavailable' error sub element.
func (el *xElement) ServiceUnavailableError() Element {
	return el.ToError(ErrServiceUnavailable.(*StanzaError))
}

// SubscriptionRequiredError returns an error copy of the element
// attaching 'subscription-required' error sub element.
func (el *xElement) SubscriptionRequiredError() Element {
	return el.ToError(ErrSubscriptionRequired.(*StanzaError))
}

// UndefinedConditionError returns an error copy of the element
// attaching 'undefined-condition' error sub element.
func (el *xElement) UndefinedConditionError() Element {
	return el.ToError(ErrUndefinedCondition.(*StanzaError))
}

// UnexpectedConditionError returns an error copy of the element
// attaching 'unexpected-condition' error sub element.
func (el *xElement) UnexpectedConditionError() Element {
	return el.ToError(ErrUnexpectedCondition.(*StanzaError))
}
