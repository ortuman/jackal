/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElement_NewElement(t *testing.T) {
	e1 := NewElementName("n")
	require.Equal(t, "n", e1.Name())
	e2 := NewElementNamespace("n", "ns")
	require.Equal(t, "n", e2.Name())
	require.Equal(t, "ns", e2.Namespace())
	e3 := NewElementFromElement(e2)
	require.Equal(t, e2.String(), e3.String())
}

func TestElement_NewError(t *testing.T) {
	e1 := NewElementNamespace("n", "ns")
	e2 := NewErrorElementFromElement(e1, ErrNotAuthorized.(*StanzaError), nil)
	require.True(t, e2.IsError())
	require.NotNil(t, e2.Error())
}

func TestElement_Gob(t *testing.T) {
	e1 := NewElementNamespace("n", "ns")
	buf := new(bytes.Buffer)
	e1.ToGob(gob.NewEncoder(buf))
	var e2 Element
	e2.FromGob(gob.NewDecoder(buf))
	require.Equal(t, e1.String(), e2.String())
}

func TestElement_Attributes(t *testing.T) {
	e1 := NewElementNamespace("n", "ns")
	e1.SetID("id")
	require.Equal(t, "id", e1.ID())
	e1.SetNamespace("ns")
	require.Equal(t, "ns", e1.Namespace())
	e1.SetLanguage("lang")
	require.Equal(t, "lang", e1.Language())
	e1.SetVersion("ver")
	require.Equal(t, "ver", e1.Version())
	e1.SetType("normal")
	require.Equal(t, "normal", e1.Type())
	e1.SetName("n2")
	require.Equal(t, "n2", e1.Name())
}

func TestElement_ToXML(t *testing.T) {
	e1 := NewElementNamespace("n", "ns")
	e1.SetID("id")
	e1.SetType("normal")
	e1.SetText("Hi!")
	e1.AppendElement(NewElementName("a"))
	e1.AppendElement(NewElementName("b"))
	buf := new(bytes.Buffer)
	e1.ToXML(buf, true)
	require.Equal(t, `<n xmlns="ns" id="id" type="normal">Hi!<a/><b/></n>`, buf.String())
	buf.Reset()
	e1.ClearElements()
	e1.SetText("")
	e1.ToXML(buf, true)
	require.Equal(t, `<n xmlns="ns" id="id" type="normal"/>`, buf.String())
	buf.Reset()
	e1.ToXML(buf, false)
	require.Equal(t, `<n xmlns="ns" id="id" type="normal">`, buf.String())
}

func TestElement_IsStanza(t *testing.T) {
	e1 := NewElementName("iq")
	e2 := NewElementName("presence")
	e3 := NewElementName("message")
	e4 := NewElementName("starttls")
	require.True(t, e1.IsStanza())
	require.True(t, e2.IsStanza())
	require.True(t, e3.IsStanza())
	require.False(t, e4.IsStanza())
}
