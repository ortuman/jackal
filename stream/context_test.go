/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestContext_Object(t *testing.T) {
	c := NewContext()
	require.Nil(t, c.Object("obj"))
	e := xmpp.NewElementName("presence")
	c.SetObject(e, "obj")
	require.Equal(t, e, c.Object("obj"))
}

func TestContext_String(t *testing.T) {
	c := NewContext()
	require.Equal(t, "", c.String("str"))
	s := "Hi world!"
	c.SetString(s, "str")
	require.Equal(t, s, c.String("str"))
}

func TestContext_Int(t *testing.T) {
	c := NewContext()
	require.Equal(t, 0, c.Int("int"))
	c.SetInt(178, "int")
	require.Equal(t, 178, c.Int("int"))
}

func TestContext_Float(t *testing.T) {
	c := NewContext()
	require.Equal(t, 0.0, c.Float("flt"))
	f := 3.141516
	c.SetFloat(f, "flt")
	require.Equal(t, f, c.Float("flt"))
}

func TestContext_Bool(t *testing.T) {
	c := NewContext()
	require.False(t, c.Bool("b"))
	c.SetBool(true, "b")
	require.True(t, c.Bool("b"))
}
