/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	require.Equal(t, badRequestErrorReason, ErrBadRequest.Error())
	require.Equal(t, conflictErrorReason, ErrConflict.Error())
	require.Equal(t, featureNotImplementedErrorReason, ErrFeatureNotImplemented.Error())
	require.Equal(t, forbiddenErrorReason, ErrForbidden.Error())
	require.Equal(t, goneErrorReason, ErrGone.Error())
	require.Equal(t, internalServerErrorErrorReason, ErrInternalServerError.Error())
	require.Equal(t, itemNotFoundErrorReason, ErrItemNotFound.Error())
	require.Equal(t, notAcceptableErrorReason, ErrNotAcceptable.Error())
	require.Equal(t, notAuthroizedErrorReason, ErrNotAuthorized.Error())
	require.Equal(t, paymentRequiredErrorReason, ErrPaymentRequired.Error())
	require.Equal(t, recipientUnavailableErrorReason, ErrRecipientUnavailable.Error())
	require.Equal(t, redirectErrorReason, ErrRedirect.Error())
	require.Equal(t, registrationRequiredErrorReason, ErrRegistrationRequired.Error())
	require.Equal(t, remoteServerNotFoundErrorReason, ErrRemoteServerNotFound.Error())
	require.Equal(t, remoteServerTimeoutErrorReason, ErrRemoteServerTimeout.Error())
	require.Equal(t, resourceConstraintErrorReason, ErrResourceConstraint.Error())
	require.Equal(t, serviceUnavailableErrorReason, ErrServiceUnavailable.Error())
	require.Equal(t, subscriptionRequiredErrorReason, ErrSubscriptionRequired.Error())
	require.Equal(t, undefinedConditionErrorReason, ErrUndefinedCondition.Error())
	require.Equal(t, unexpectedConditionErrorReason, ErrUnexpectedCondition.Error())

	e := NewElementName("elem")
	require.NotNil(t, e.BadRequestError().Error().FindElement(badRequestErrorReason))
	require.NotNil(t, e.ConflictError().Error().FindElement(conflictErrorReason))
	require.NotNil(t, e.FeatureNotImplementedError().Error().FindElement(featureNotImplementedErrorReason))
	require.NotNil(t, e.ForbiddenError().Error().FindElement(forbiddenErrorReason))
	require.NotNil(t, e.GoneError().Error().FindElement(goneErrorReason))
	require.NotNil(t, e.InternalServerError().Error().FindElement(internalServerErrorErrorReason))
	require.NotNil(t, e.ItemNotFoundError().Error().FindElement(itemNotFoundErrorReason))
	require.NotNil(t, e.JidMalformedError().Error().FindElement(jidMalformedErrorReason))
	require.NotNil(t, e.NotAcceptableError().Error().FindElement(notAcceptableErrorReason))
	require.NotNil(t, e.NotAllowedError().Error().FindElement(notAllowedErrorReason))
	require.NotNil(t, e.NotAuthorizedError().Error().FindElement(notAuthroizedErrorReason))
	require.NotNil(t, e.PaymentRequiredError().Error().FindElement(paymentRequiredErrorReason))
	require.NotNil(t, e.RecipientUnavailableError().Error().FindElement(recipientUnavailableErrorReason))
	require.NotNil(t, e.RedirectError().Error().FindElement(redirectErrorReason))
	require.NotNil(t, e.RegistrationRequiredError().Error().FindElement(registrationRequiredErrorReason))
	require.NotNil(t, e.RemoteServerNotFoundError().Error().FindElement(remoteServerNotFoundErrorReason))
	require.NotNil(t, e.RemoteServerTimeoutError().Error().FindElement(remoteServerTimeoutErrorReason))
	require.NotNil(t, e.ResourceConstraintError().Error().FindElement(resourceConstraintErrorReason))
	require.NotNil(t, e.ServiceUnavailableError().Error().FindElement(serviceUnavailableErrorReason))
	require.NotNil(t, e.SubscriptionRequiredError().Error().FindElement(subscriptionRequiredErrorReason))
	require.NotNil(t, e.UndefinedConditionError().Error().FindElement(undefinedConditionErrorReason))
	require.NotNil(t, e.UnexpectedConditionError().Error().FindElement(unexpectedConditionErrorReason))
}
