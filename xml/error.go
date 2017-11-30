/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "strconv"

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
	errBadRequest            = newErrorElement(400, modifyErrorType, badRequestErrorReason)
	errConflict              = newErrorElement(409, cancelErrorType, conflictErrorReason)
	errFeatureNotImplemented = newErrorElement(501, cancelErrorType, featureNotImplementedErrorReason)
	errForbidden             = newErrorElement(403, authErrorType, forbiddenErrorReason)
	errGone                  = newErrorElement(302, modifyErrorType, goneErrorReason)
	errInternalServerError   = newErrorElement(500, waitErrorType, internalServerErrorErrorReason)
	errItemNotFound          = newErrorElement(404, cancelErrorType, itemNotFoundErrorReason)
	errJidMalformed          = newErrorElement(400, modifyErrorType, jidMalformedErrorReason)
	errNotAcceptable         = newErrorElement(406, modifyErrorType, notAcceptableErrorReason)
	errNotAllowed            = newErrorElement(405, cancelErrorType, notAllowedErrorReason)
	errNotAuthorized         = newErrorElement(405, authErrorType, notAuthroizedErrorReason)
	errPaymentRequired       = newErrorElement(402, authErrorType, paymentRequiredErrorReason)
	errRecipientUnavailable  = newErrorElement(404, waitErrorType, recipientUnavailableErrorReason)
	errRedirect              = newErrorElement(302, modifyErrorType, redirectErrorReason)
	errRegistrationRequired  = newErrorElement(407, authErrorType, registrationRequiredErrorReason)
	errRemoteServerNotFound  = newErrorElement(404, cancelErrorType, remoteServerNotFoundErrorReason)
	errRemoteServerTimeout   = newErrorElement(504, waitErrorType, remoteServerTimeoutErrorReason)
	errResourceConstraint    = newErrorElement(500, waitErrorType, resourceConstraintErrorReason)
	errServiceUnavailable    = newErrorElement(503, cancelErrorType, serviceUnavailableErrorReason)
	errSubscriptionRequired  = newErrorElement(407, authErrorType, subscriptionRequiredErrorReason)
	errUndefinedCondition    = newErrorElement(500, waitErrorType, undefinedConditionErrorReason)
	errUnexpectedCondition   = newErrorElement(400, waitErrorType, unexpectedConditionErrorReason)
)

func newErrorElement(code int, errorType string, reason string) *Element {
	err := NewMutableElementName("error")
	err.SetAttribute("code", strconv.Itoa(code))
	err.SetAttribute("type", errorType)
	err.AppendElement(NewElementNamespace(reason, "urn:ietf:params:xml:ns:xmpp-stanzas"))
	return err.Copy()
}

// BadRequestError - the sender has sent XML that is malformed
// or that cannot be processed.
func (e *Element) BadRequestError() *Element {
	return e.toErrorElement(errBadRequest)
}

// ConflictError - access cannot be granted because an existing resource
// or session exists with the same name or address.
func (e *Element) ConflictError() *Element {
	return e.toErrorElement(errConflict)
}

// FeatureNotImplementedError - the feature requested is not implemented by the server
// and therefore cannot be processed.
func (e *Element) FeatureNotImplementedError() *Element {
	return e.toErrorElement(errFeatureNotImplemented)
}

// ForbiddenError - the requesting entity does not possess the required permissions to perform the action.
func (e *Element) ForbiddenError() *Element {
	return e.toErrorElement(errForbidden)
}

// GoneError - the recipient or server can no longer be contacted at this address
func (e *Element) GoneError() *Element {
	return e.toErrorElement(errGone)
}

// InternalServerError - the server could not process the stanza because of a misconfiguration
// or an otherwise-undefined internal server error.
func (e *Element) InternalServerError() *Element {
	return e.toErrorElement(errInternalServerError)
}

// ItemNotFoundError - the addressed JID or item requested cannot be found.
func (e *Element) ItemNotFoundError() *Element {
	return e.toErrorElement(errItemNotFound)
}

// JidMalformedError - the sending entity has provided or communicated an XMPP address or aspect thereof
// that does not adhere to the syntax defined in https://xmpp.org/rfcs/rfc3920.html#addressing.
func (e *Element) JidMalformedError() *Element {
	return e.toErrorElement(errJidMalformed)
}

// NotAcceptableError - the server understands the request but is refusing
// to process it because it does not meet the defined criteria.
func (e *Element) NotAcceptableError() *Element {
	return e.toErrorElement(errNotAcceptable)
}

// NotAllowedError - the recipient or server does not allow any entity to perform the action.
func (e *Element) NotAllowedError() *Element {
	return e.toErrorElement(errNotAllowed)
}

// NotAuthorizedError - the sender must provide proper credentials before being allowed to perform the action,
// or has provided improper credentials.
func (e *Element) NotAuthorizedError() *Element {
	return e.toErrorElement(errNotAuthorized)
}

// PaymentRequiredError - the requesting entity is not authorized to access
// the requested service because payment is required.
func (e *Element) PaymentRequiredError() *Element {
	return e.toErrorElement(errPaymentRequired)
}

// RecipientUnavailableError - the intended recipient is temporarily unavailable.
func (e *Element) RecipientUnavailableError() *Element {
	return e.toErrorElement(errRecipientUnavailable)
}

// RedirectError - the recipient or server is redirecting requests for this information
// to another entity, usually temporarily.
func (e *Element) RedirectError() *Element {
	return e.toErrorElement(errRedirect)
}

// RegistrationRequiredError - the requesting entity is not authorized to access
// the requested service because registration is required.
func (e *Element) RegistrationRequiredError() *Element {
	return e.toErrorElement(errRegistrationRequired)
}

// RemoteServerNotFoundError - a remote server or service specified as part or all of the JID
// of the intended recipient does not exist.
func (e *Element) RemoteServerNotFoundError() *Element {
	return e.toErrorElement(errRemoteServerNotFound)
}

// RemoteServerTimeoutError - a remote server or service specified as part or all of the JID
// of the intended recipient could not be contacted within a reasonable amount of time.
func (e *Element) RemoteServerTimeoutError() *Element {
	return e.toErrorElement(errRemoteServerTimeout)
}

// ResourceConstraintError - the server or recipient lacks the system resources
// necessary to service the request.
func (e *Element) ResourceConstraintError() *Element {
	return e.toErrorElement(errResourceConstraint)
}

// ServiceUnavailableError - the recipient or server or recipient does not currently
// provide the requested service.
func (e *Element) ServiceUnavailableError() *Element {
	return e.toErrorElement(errServiceUnavailable)
}

// SubscriptionRequiredError - the requesting entity is not authorized to
// access the requested service because a subscription is required.
func (e *Element) SubscriptionRequiredError() *Element {
	return e.toErrorElement(errSubscriptionRequired)
}

// UndefinedConditionError - the error condition is not one of those defined
// by the other conditions in this list.
func (e *Element) UndefinedConditionError() *Element {
	return e.toErrorElement(errUndefinedCondition)
}

// UnexpectedConditionError - the recipient or server understood the request
// but was not expecting it at this time.
func (e *Element) UnexpectedConditionError() *Element {
	return e.toErrorElement(errUnexpectedCondition)
}

func (e *Element) toErrorElement(errElement *Element) *Element {
	ret := NewMutableElement(e)
	ret.SetAttribute("type", "error")
	ret.AppendElement(errElement)
	return ret.Copy()
}
