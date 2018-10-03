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

func TestDataForm_FromElement(t *testing.T) {
	elem := xmpp.NewElementName("x1")
	_, err := NewFormFromElement(elem)
	require.NotNil(t, err)

	elem.SetName("x")
	_, err = NewFormFromElement(elem)
	require.NotNil(t, err)

	elem.SetNamespace(formNamespace)
	_, err = NewFormFromElement(elem)
	require.NotNil(t, err)

	elem.SetAttribute("type", Form)
	_, err = NewFormFromElement(elem)
	require.Nil(t, err)

	titleElem := xmpp.NewElementName("title")
	titleElem.SetText("A title")
	instElem := xmpp.NewElementName("instructions")
	instElem.SetText("A set of instructions")
	elem.AppendElement(titleElem)
	elem.AppendElement(instElem)
	form, _ := NewFormFromElement(elem)
	require.NotNil(t, form)
	require.Equal(t, "A title", form.Title)
	require.Equal(t, "A set of instructions", form.Instructions)

	fieldElem := xmpp.NewElementName("field")
	fieldElem.SetAttribute("var", "vn")
	fieldElem.SetAttribute("type", Boolean)

	reportedElem := xmpp.NewElementName("reported")
	reportedElem.AppendElement(fieldElem)
	elem.AppendElement(reportedElem)

	itemElem := xmpp.NewElementName("item")
	itemElem.AppendElement(fieldElem)
	elem.AppendElement(itemElem)

	elem.AppendElement(fieldElem)

	form, _ = NewFormFromElement(elem)
	require.NotNil(t, form)
	require.Equal(t, 1, len(form.Reported))
	require.Equal(t, 1, len(form.Items))
	require.Equal(t, 1, len(form.Fields))
}

func TestDataForm_Element(t *testing.T) {
	form := &DataForm{}
	form.Type = Form
	elem := form.Element()
	require.Equal(t, "x", elem.Name())
	require.Equal(t, formNamespace, elem.Namespace())

	form.Title = "A title"
	form.Instructions = "A set of instructions"
	elem = form.Element()

	titleElem := elem.Elements().Child("title")
	instElem := elem.Elements().Child("instructions")
	require.NotNil(t, titleElem)
	require.NotNil(t, instElem)
	require.Equal(t, "A title", titleElem.Text())
	require.Equal(t, "A set of instructions", instElem.Text())

	form.Reported = []Field{{Var: "var1"}}
	form.Items = [][]Field{{{Var: "var2"}}}

	elem = form.Element()
	require.NotNil(t, elem.Elements().Child("reported"))
	require.Equal(t, 1, len(elem.Elements().Children("item")))
}
