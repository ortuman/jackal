/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0004

import (
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestField_FromElement(t *testing.T) {
	elem := xmpp.NewElementName("")
	_, err := NewFieldFromElement(elem)
	require.NotNil(t, err)

	elem.SetName("field")
	elem.SetAttribute("var", "name")
	elem.SetAttribute("type", "integer")
	_, err = NewFieldFromElement(elem)
	require.NotNil(t, err)

	elem.SetAttribute("type", TextSingle)
	_, err = NewFieldFromElement(elem)
	require.Nil(t, err)

	desc := "A description"
	descElem := xmpp.NewElementName("desc")
	descElem.SetText(desc)
	elem.AppendElement(descElem)
	elem.AppendElement(xmpp.NewElementName("required"))
	f, err := NewFieldFromElement(elem)
	require.Nil(t, err)
	require.Equal(t, desc, f.Description)
	require.True(t, f.Required)

	value := "A value"
	valueElem := xmpp.NewElementName("value")
	valueElem.SetText(value)
	elem.AppendElement(valueElem)
	f, err = NewFieldFromElement(elem)
	require.Nil(t, err)
	require.Equal(t, 1, len(f.Values))
	require.Equal(t, value, f.Values[0])
	elem.RemoveElements("value")

	optValue := "An option value"
	valueElem.SetText(optValue)
	optElem := xmpp.NewElementName("option")
	optElem.SetAttribute("label", "news")
	optElem.AppendElement(valueElem)
	elem.AppendElement(optElem)
	f, err = NewFieldFromElement(elem)
	require.Nil(t, err)
	require.Equal(t, 1, len(f.Options))
	require.Equal(t, "news", f.Options[0].Label)
	require.Equal(t, optValue, f.Options[0].Value)
}

func TestField_Element(t *testing.T) {
	f := Field{Var: "a_var"}
	f.Type = "a_type"
	f.Label = "a_label"
	f.Required = true
	f.Description = "A description"
	f.Values = []string{"A value"}
	f.Options = []Option{{"opt_label", "An option value"}}
	elem := f.Element()

	require.Equal(t, "field", elem.Name())
	require.Equal(t, "a_var", elem.Attributes().Get("var"))
	require.Equal(t, "a_type", elem.Attributes().Get("type"))
	require.Equal(t, "a_label", elem.Attributes().Get("label"))

	valElem := elem.Elements().Child("value")
	require.NotNil(t, valElem)
	require.Equal(t, "A value", valElem.Text())

	optElems := elem.Elements().Children("option")
	require.Equal(t, 1, len(optElems))
	optElem := optElems[0]
	require.Equal(t, "opt_label", optElem.Attributes().Get("label"))

	valElem = optElem.Elements().Child("value")
	require.Equal(t, "An option value", valElem.Text())
}
