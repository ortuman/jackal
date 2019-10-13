/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
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
	require.Equal(t, unexpectedRequestErrorReason, ErrUnexpectedRequest.Error())

	j, _ := jid.New("", "jackal.im", "", true)
	e := NewIQType(uuid.New(), GetType)
	e.SetFromJID(j)
	e.SetToJID(j)

	require.NotNil(t, e.BadRequestError().Error().Elements().Child(badRequestErrorReason))
	require.NotNil(t, e.ConflictError().Error().Elements().Child(conflictErrorReason))
	require.NotNil(t, e.FeatureNotImplementedError().Error().Elements().Child(featureNotImplementedErrorReason))
	require.NotNil(t, e.ForbiddenError().Error().Elements().Child(forbiddenErrorReason))
	require.NotNil(t, e.GoneError().Error().Elements().Child(goneErrorReason))
	require.NotNil(t, e.InternalServerError().Error().Elements().Child(internalServerErrorErrorReason))
	require.NotNil(t, e.ItemNotFoundError().Error().Elements().Child(itemNotFoundErrorReason))
	require.NotNil(t, e.JidMalformedError().Error().Elements().Child(jidMalformedErrorReason))
	require.NotNil(t, e.NotAcceptableError().Error().Elements().Child(notAcceptableErrorReason))
	require.NotNil(t, e.NotAllowedError().Error().Elements().Child(notAllowedErrorReason))
	require.NotNil(t, e.NotAuthorizedError().Error().Elements().Child(notAuthroizedErrorReason))
	require.NotNil(t, e.PaymentRequiredError().Error().Elements().Child(paymentRequiredErrorReason))
	require.NotNil(t, e.RecipientUnavailableError().Error().Elements().Child(recipientUnavailableErrorReason))
	require.NotNil(t, e.RedirectError().Error().Elements().Child(redirectErrorReason))
	require.NotNil(t, e.RegistrationRequiredError().Error().Elements().Child(registrationRequiredErrorReason))
	require.NotNil(t, e.RemoteServerNotFoundError().Error().Elements().Child(remoteServerNotFoundErrorReason))
	require.NotNil(t, e.RemoteServerTimeoutError().Error().Elements().Child(remoteServerTimeoutErrorReason))
	require.NotNil(t, e.ResourceConstraintError().Error().Elements().Child(resourceConstraintErrorReason))
	require.NotNil(t, e.ServiceUnavailableError().Error().Elements().Child(serviceUnavailableErrorReason))
	require.NotNil(t, e.SubscriptionRequiredError().Error().Elements().Child(subscriptionRequiredErrorReason))
	require.NotNil(t, e.UndefinedConditionError().Error().Elements().Child(undefinedConditionErrorReason))
	require.NotNil(t, e.UnexpectedConditionError().Error().Elements().Child(unexpectedConditionErrorReason))
	require.NotNil(t, e.UnexpectedRequestError().Error().Elements().Child(unexpectedRequestErrorReason))
}
