/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import "strconv"

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

func (se *StanzaError) Error() string {
	return se.reason
}

func (se *StanzaError) Element() *XElement {
	err := NewElementName("error")
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
	ErrBadRequest            = newErrorElement(400, modifyErrorType, badRequestErrorReason)
	ErrConflict              = newErrorElement(409, cancelErrorType, conflictErrorReason)
	ErrFeatureNotImplemented = newErrorElement(501, cancelErrorType, featureNotImplementedErrorReason)
	ErrForbidden             = newErrorElement(403, authErrorType, forbiddenErrorReason)
	ErrGone                  = newErrorElement(302, modifyErrorType, goneErrorReason)
	ErrInternalServerError   = newErrorElement(500, waitErrorType, internalServerErrorErrorReason)
	ErrItemNotFound          = newErrorElement(404, cancelErrorType, itemNotFoundErrorReason)
	ErrJidMalformed          = newErrorElement(400, modifyErrorType, jidMalformedErrorReason)
	ErrNotAcceptable         = newErrorElement(406, modifyErrorType, notAcceptableErrorReason)
	ErrNotAllowed            = newErrorElement(405, cancelErrorType, notAllowedErrorReason)
	ErrNotAuthorized         = newErrorElement(405, authErrorType, notAuthroizedErrorReason)
	ErrPaymentRequired       = newErrorElement(402, authErrorType, paymentRequiredErrorReason)
	ErrRecipientUnavailable  = newErrorElement(404, waitErrorType, recipientUnavailableErrorReason)
	ErrRedirect              = newErrorElement(302, modifyErrorType, redirectErrorReason)
	ErrRegistrationRequired  = newErrorElement(407, authErrorType, registrationRequiredErrorReason)
	ErrRemoteServerNotFound  = newErrorElement(404, cancelErrorType, remoteServerNotFoundErrorReason)
	ErrRemoteServerTimeout   = newErrorElement(504, waitErrorType, remoteServerTimeoutErrorReason)
	ErrResourceConstraint    = newErrorElement(500, waitErrorType, resourceConstraintErrorReason)
	ErrServiceUnavailable    = newErrorElement(503, cancelErrorType, serviceUnavailableErrorReason)
	ErrSubscriptionRequired  = newErrorElement(407, authErrorType, subscriptionRequiredErrorReason)
	ErrUndefinedCondition    = newErrorElement(500, waitErrorType, undefinedConditionErrorReason)
	ErrUnexpectedCondition   = newErrorElement(400, waitErrorType, unexpectedConditionErrorReason)
)

// BadRequestError - the sender has sent XML that is malformed
// or that cannot be processed.
func (e *XElement) BadRequestError() *XElement {
	return e.ToError(ErrBadRequest.(*StanzaError))
}

// ConflictError - access cannot be granted because an existing resource
// or session exists with the same name or address.
func (e *XElement) ConflictError() *XElement {
	return e.ToError(ErrConflict.(*StanzaError))
}

// FeatureNotImplementedError - the feature requested is not implemented by the server
// and therefore cannot be processed.
func (e *XElement) FeatureNotImplementedError() *XElement {
	return e.ToError(ErrFeatureNotImplemented.(*StanzaError))
}

// ForbiddenError - the requesting entity does not possess the required permissions to perform the action.
func (e *XElement) ForbiddenError() *XElement {
	return e.ToError(ErrForbidden.(*StanzaError))
}

// GoneError - the recipient or server can no longer be contacted at this address
func (e *XElement) GoneError() *XElement {
	return e.ToError(ErrGone.(*StanzaError))
}

// InternalServerError - the server could not process the stanza because of a misconfiguration
// or an otherwise-undefined internal server error.
func (e *XElement) InternalServerError() *XElement {
	return e.ToError(ErrInternalServerError.(*StanzaError))
}

// ItemNotFoundError - the addressed JID or item requested cannot be found.
func (e *XElement) ItemNotFoundError() *XElement {
	return e.ToError(ErrItemNotFound.(*StanzaError))
}

// JidMalformedError - the sending entity has provided or communicated an XMPP address or aspect thereof
// that does not adhere to the syntax defined in https://xmpp.org/rfcs/rfc3920.html#addressing.
func (e *XElement) JidMalformedError() *XElement {
	return e.ToError(ErrJidMalformed.(*StanzaError))
}

// NotAcceptableError - the server understands the request but is refusing
// to process it because it does not meet the defined criteria.
func (e *XElement) NotAcceptableError() *XElement {
	return e.ToError(ErrNotAcceptable.(*StanzaError))
}

// NotAllowedError - the recipient or server does not allow any entity to perform the action.
func (e *XElement) NotAllowedError() *XElement {
	return e.ToError(ErrNotAllowed.(*StanzaError))
}

// NotAuthorizedError - the sender must provide proper credentials before being allowed to perform the action,
// or has provided improper credentials.
func (e *XElement) NotAuthorizedError() *XElement {
	return e.ToError(ErrNotAuthorized.(*StanzaError))
}

// PaymentRequiredError - the requesting entity is not authorized to access
// the requested service because payment is required.
func (e *XElement) PaymentRequiredError() *XElement {
	return e.ToError(ErrPaymentRequired.(*StanzaError))
}

// RecipientUnavailableError - the intended recipient is temporarily unavailable.
func (e *XElement) RecipientUnavailableError() *XElement {
	return e.ToError(ErrRecipientUnavailable.(*StanzaError))
}

// RedirectError - the recipient or server is redirecting requests for this information
// to another entity, usually temporarily.
func (e *XElement) RedirectError() *XElement {
	return e.ToError(ErrRedirect.(*StanzaError))
}

// RegistrationRequiredError - the requesting entity is not authorized to access
// the requested service because registration is required.
func (e *XElement) RegistrationRequiredError() *XElement {
	return e.ToError(ErrRegistrationRequired.(*StanzaError))
}

// RemoteServerNotFoundError - a remote server or service specified as part or all of the JID
// of the intended recipient does not exist.
func (e *XElement) RemoteServerNotFoundError() *XElement {
	return e.ToError(ErrRemoteServerNotFound.(*StanzaError))
}

// RemoteServerTimeoutError - a remote server or service specified as part or all of the JID
// of the intended recipient could not be contacted within a reasonable amount of time.
func (e *XElement) RemoteServerTimeoutError() *XElement {
	return e.ToError(ErrRemoteServerTimeout.(*StanzaError))
}

// ResourceConstraintError - the server or recipient lacks the system resources
// necessary to service the request.
func (e *XElement) ResourceConstraintError() *XElement {
	return e.ToError(ErrResourceConstraint.(*StanzaError))
}

// ServiceUnavailableError - the recipient or server or recipient does not currently
// provide the requested service.
func (e *XElement) ServiceUnavailableError() *XElement {
	return e.ToError(ErrServiceUnavailable.(*StanzaError))
}

// SubscriptionRequiredError - the requesting entity is not authorized to
// access the requested service because a subscription is required.
func (e *XElement) SubscriptionRequiredError() *XElement {
	return e.ToError(ErrSubscriptionRequired.(*StanzaError))
}

// UndefinedConditionError - the error condition is not one of those defined
// by the other conditions in this list.
func (e *XElement) UndefinedConditionError() *XElement {
	return e.ToError(ErrUndefinedCondition.(*StanzaError))
}

// UnexpectedConditionError - the recipient or server understood the request
// but was not expecting it at this time.
func (e *XElement) UnexpectedConditionError() *XElement {
	return e.ToError(ErrUnexpectedCondition.(*StanzaError))
}

func (e *XElement) ToError(stanzaError *StanzaError) *XElement {
	e.SetAttribute("type", "error")
	e.AppendElement(stanzaError.Element())
	return e
}
