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
func (el *XElement) BadRequestError() *XElement {
	return ToErrorElement(el, ErrBadRequest.(*StanzaError))
}

// ConflictError - access cannot be granted because an existing resource
// or session exists with the same name or address.
func (el *XElement) ConflictError() *XElement {
	return ToErrorElement(el, ErrConflict.(*StanzaError))
}

// FeatureNotImplementedError - the feature requested is not implemented by the server
// and therefore cannot be processed.
func (el *XElement) FeatureNotImplementedError() *XElement {
	return ToErrorElement(el, ErrFeatureNotImplemented.(*StanzaError))
}

// ForbiddenError - the requesting entity does not possess the required permissions to perform the action.
func (el *XElement) ForbiddenError() *XElement {
	return ToErrorElement(el, ErrForbidden.(*StanzaError))
}

// GoneError - the recipient or server can no longer be contacted at this address
func (el *XElement) GoneError() *XElement {
	return ToErrorElement(el, ErrGone.(*StanzaError))
}

// InternalServerError - the server could not process the stanza because of a misconfiguration
// or an otherwise-undefined internal server error.
func (el *XElement) InternalServerError() *XElement {
	return ToErrorElement(el, ErrInternalServerError.(*StanzaError))
}

// ItemNotFoundError - the addressed JID or item requested cannot be found.
func (el *XElement) ItemNotFoundError() *XElement {
	return ToErrorElement(el, ErrItemNotFound.(*StanzaError))
}

// JidMalformedError - the sending entity has provided or communicated an XMPP address or aspect thereof
// that does not adhere to the syntax defined in https://xmpp.org/rfcs/rfc3920.html#addressing.
func (el *XElement) JidMalformedError() *XElement {
	return ToErrorElement(el, ErrJidMalformed.(*StanzaError))
}

// NotAcceptableError - the server understands the request but is refusing
// to process it because it does not meet the defined criteria.
func (el *XElement) NotAcceptableError() *XElement {
	return ToErrorElement(el, ErrNotAcceptable.(*StanzaError))
}

// NotAllowedError - the recipient or server does not allow any entity to perform the action.
func (el *XElement) NotAllowedError() *XElement {
	return ToErrorElement(el, ErrNotAllowed.(*StanzaError))
}

// NotAuthorizedError - the sender must provide proper credentials before being allowed to perform the action,
// or has provided improper credentials.
func (el *XElement) NotAuthorizedError() *XElement {
	return ToErrorElement(el, ErrNotAuthorized.(*StanzaError))
}

// PaymentRequiredError - the requesting entity is not authorized to access
// the requested service because payment is required.
func (el *XElement) PaymentRequiredError() *XElement {
	return ToErrorElement(el, ErrPaymentRequired.(*StanzaError))
}

// RecipientUnavailableError - the intended recipient is temporarily unavailable.
func (el *XElement) RecipientUnavailableError() *XElement {
	return ToErrorElement(el, ErrRecipientUnavailable.(*StanzaError))
}

// RedirectError - the recipient or server is redirecting requests for this information
// to another entity, usually temporarily.
func (el *XElement) RedirectError() *XElement {
	return ToErrorElement(el, ErrRedirect.(*StanzaError))
}

// RegistrationRequiredError - the requesting entity is not authorized to access
// the requested service because registration is required.
func (el *XElement) RegistrationRequiredError() *XElement {
	return ToErrorElement(el, ErrRegistrationRequired.(*StanzaError))
}

// RemoteServerNotFoundError - a remote server or service specified as part or all of the JID
// of the intended recipient does not exist.
func (el *XElement) RemoteServerNotFoundError() *XElement {
	return ToErrorElement(el, ErrRemoteServerNotFound.(*StanzaError))
}

// RemoteServerTimeoutError - a remote server or service specified as part or all of the JID
// of the intended recipient could not be contacted within a reasonable amount of time.
func (el *XElement) RemoteServerTimeoutError() *XElement {
	return ToErrorElement(el, ErrRemoteServerTimeout.(*StanzaError))
}

// ResourceConstraintError - the server or recipient lacks the system resources
// necessary to service the request.
func (el *XElement) ResourceConstraintError() *XElement {
	return ToErrorElement(el, ErrResourceConstraint.(*StanzaError))
}

// ServiceUnavailableError - the recipient or server or recipient does not currently
// provide the requested service.
func (el *XElement) ServiceUnavailableError() *XElement {
	return ToErrorElement(el, ErrServiceUnavailable.(*StanzaError))
}

// SubscriptionRequiredError - the requesting entity is not authorized to
// access the requested service because a subscription is required.
func (el *XElement) SubscriptionRequiredError() *XElement {
	return ToErrorElement(el, ErrSubscriptionRequired.(*StanzaError))
}

// UndefinedConditionError - the error condition is not one of those defined
// by the other conditions in this list.
func (el *XElement) UndefinedConditionError() *XElement {
	return ToErrorElement(el, ErrUndefinedCondition.(*StanzaError))
}

// UnexpectedConditionError - the recipient or server understood the request
// but was not expecting it at this time.
func (el *XElement) UnexpectedConditionError() *XElement {
	return ToErrorElement(el, ErrUnexpectedCondition.(*StanzaError))
}

func ToErrorElement(elem Element, stanzaError *StanzaError) *XElement {
	errEl := &XElement{
		name:     elem.Name(),
		text:     elem.Text(),
		attrs:    elem.Attributes(),
		elements: elem.Elements(),
	}
	errEl.SetAttribute("type", "error")
	errEl.AppendElement(stanzaError.Element())
	return errEl
}
