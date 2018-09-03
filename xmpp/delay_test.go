/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp_test

import (
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestDelay(t *testing.T) {
	e := xmpp.NewElementName("element")
	e.Delay("example.org", "any text")
	delay := e.Elements().Child("delay")
	require.NotNil(t, delay)
	require.Equal(t, "example.org", delay.Attributes().Get("from"))
	require.Equal(t, "any text", delay.Text())
}
