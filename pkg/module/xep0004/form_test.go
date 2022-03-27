// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0004

import (
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/stretchr/testify/require"
)

func TestDataForm_FromElementError(t *testing.T) {
	// given
	eb := stravaganza.NewBuilder("x1")

	// then
	_, err := NewFormFromElement(eb.Build())
	require.NotNil(t, err)
}

func TestDataForm_FromElementMissingNamespace(t *testing.T) {
	// given
	eb := stravaganza.NewBuilder("x")

	// then
	_, err := NewFormFromElement(eb.Build())
	require.NotNil(t, err)
}

func TestDataForm_FromElementSuccess(t *testing.T) {
	// given
	eb := stravaganza.NewBuilder("x")
	eb.WithAttribute(stravaganza.Namespace, FormNamespace)
	eb.WithAttribute("type", Form)

	// when
	f, err := NewFormFromElement(eb.Build())

	// then
	require.Nil(t, err)
	require.NotNil(t, f)
}

func TestDataForm_FromElementForm(t *testing.T) {
	eb := stravaganza.NewBuilder("x")
	eb.WithAttribute(stravaganza.Namespace, FormNamespace)
	eb.WithAttribute("type", Form)

	titleB := stravaganza.NewBuilder("title")
	titleB.WithText("A title")
	instB := stravaganza.NewBuilder("instructions")
	instB.WithText("A set of instructions")
	eb.WithChild(titleB.Build())
	eb.WithChild(instB.Build())

	form, _ := NewFormFromElement(eb.Build())

	require.NotNil(t, form)
	require.Equal(t, "A title", form.Title)
	require.Equal(t, "A set of instructions", form.Instructions)

	fieldB := stravaganza.NewBuilder("field")
	fieldB.WithAttribute("var", "vn")
	fieldB.WithAttribute("type", Boolean)

	reportedB := stravaganza.NewBuilder("reported")
	reportedB.WithChild(fieldB.Build())
	eb.WithChild(reportedB.Build())

	itemB := stravaganza.NewBuilder("item")
	itemB.WithChild(fieldB.Build())
	eb.WithChild(itemB.Build())

	eb.WithChild(fieldB.Build())

	form, _ = NewFormFromElement(eb.Build())

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
	require.Equal(t, FormNamespace, elem.Attribute(stravaganza.Namespace))

	form.Title = "A title"
	form.Instructions = "A set of instructions"
	elem = form.Element()

	titleElem := elem.Child("title")
	instElem := elem.Child("instructions")
	require.NotNil(t, titleElem)
	require.NotNil(t, instElem)
	require.Equal(t, "A title", titleElem.Text())
	require.Equal(t, "A set of instructions", instElem.Text())

	form.Reported = []Field{{Var: "var1"}}
	form.Items = []Fields{{{Var: "var2"}}}

	elem = form.Element()
	require.NotNil(t, elem.Child("reported"))
	require.Equal(t, 1, len(elem.Children("item")))
}
