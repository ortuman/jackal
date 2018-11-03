/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package streamerror

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamError(t *testing.T) {
	require.Equal(t, "invalid-xml", ErrInvalidXML.Error())
	require.Equal(t, "invalid-xml", ErrInvalidXML.Element().Elements().All()[0].Name())

	require.Equal(t, "invalid-namespace", ErrInvalidNamespace.Error())
	require.Equal(t, "invalid-namespace", ErrInvalidNamespace.Element().Elements().All()[0].Name())

	require.Equal(t, "host-unknown", ErrHostUnknown.Error())
	require.Equal(t, "host-unknown", ErrHostUnknown.Element().Elements().All()[0].Name())

	require.Equal(t, "invalid-from", ErrInvalidFrom.Error())
	require.Equal(t, "invalid-from", ErrInvalidFrom.Element().Elements().All()[0].Name())

	require.Equal(t, "connection-timeout", ErrConnectionTimeout.Error())
	require.Equal(t, "connection-timeout", ErrConnectionTimeout.Element().Elements().All()[0].Name())

	require.Equal(t, "unsupported-stanza-type", ErrUnsupportedStanzaType.Error())
	require.Equal(t, "unsupported-stanza-type", ErrUnsupportedStanzaType.Element().Elements().All()[0].Name())

	require.Equal(t, "unsupported-version", ErrUnsupportedVersion.Error())
	require.Equal(t, "unsupported-version", ErrUnsupportedVersion.Element().Elements().All()[0].Name())

	require.Equal(t, "not-authorized", ErrNotAuthorized.Error())
	require.Equal(t, "not-authorized", ErrNotAuthorized.Element().Elements().All()[0].Name())

	require.Equal(t, "system-shutdown", ErrSystemShutdown.Error())
	require.Equal(t, "system-shutdown", ErrSystemShutdown.Element().Elements().All()[0].Name())

	require.Equal(t, "resource-constraint", ErrResourceConstraint.Error())
	require.Equal(t, "resource-constraint", ErrResourceConstraint.Element().Elements().All()[0].Name())

	require.Equal(t, "internal-server-error", ErrInternalServerError.Error())
	require.Equal(t, "internal-server-error", ErrInternalServerError.Element().Elements().All()[0].Name())
}
