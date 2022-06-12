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

func TestField_FromElementError(t *testing.T) {
	// given
	_, err := NewFieldFromElement(stravaganza.NewBuilder("").Build())

	// then
	require.NotNil(t, err)
}

func TestField_FromElementInvalidField(t *testing.T) {
	// given
	eb := stravaganza.NewBuilder("field")
	eb.WithAttribute("var", "name")
	eb.WithAttribute("type", "integer")

	// when
	f, err := NewFieldFromElement(eb.Build())

	// then
	require.NotNil(t, err)
	require.Nil(t, f)
}

func TestField_FromElementDescription(t *testing.T) {
	// given
	eb := stravaganza.NewBuilder("field")
	eb.WithAttribute("var", "name")
	eb.WithAttribute("type", TextSingle)

	desc := "A description"
	descB := stravaganza.NewBuilder("desc")
	descB.WithText(desc)
	eb.WithChild(descB.Build())
	eb.WithChild(stravaganza.NewBuilder("required").Build())

	// when
	f, err := NewFieldFromElement(eb.Build())

	// then
	require.Nil(t, err)
	require.Equal(t, desc, f.Description)
	require.True(t, f.Required)
}

func TestField_FromElementValue(t *testing.T) {
	// given
	eb := stravaganza.NewBuilder("field")
	eb.WithAttribute("var", "name")
	eb.WithAttribute("type", TextSingle)

	value := "A value"
	valueB := stravaganza.NewBuilder("value")
	valueB.WithText(value)
	eb.WithChild(valueB.Build())

	// when
	f, err := NewFieldFromElement(eb.Build())

	// then
	require.Nil(t, err)
	require.Equal(t, 1, len(f.Values))
	require.Equal(t, value, f.Values[0])
}

func TestField_FromElementOptValue(t *testing.T) {
	// given
	eb := stravaganza.NewBuilder("field")
	eb.WithAttribute("var", "name")
	eb.WithAttribute("type", TextSingle)

	optValue := "An option value"
	valueB := stravaganza.NewBuilder("value")
	valueB.WithText(optValue)
	optElem := stravaganza.NewBuilder("option")
	optElem.WithAttribute("label", "news")
	optElem.WithChild(valueB.Build())
	eb.WithChild(optElem.Build())

	// when
	f, err := NewFieldFromElement(eb.Build())

	// then
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
	f.Validate = &Validate{
		DataType: BooleanDataType,
		Validator: &RegExValidator{
			RegEx: "([0-9]{3})-([0-9]{2})-([0-9]{4})",
		},
	}
	elem := f.Element()

	require.Equal(t, "field", elem.Name())
	require.Equal(t, "a_var", elem.Attribute("var"))
	require.Equal(t, "a_type", elem.Attribute("type"))
	require.Equal(t, "a_label", elem.Attribute("label"))

	valElem := elem.Child("value")
	require.NotNil(t, valElem)
	require.Equal(t, "A value", valElem.Text())

	optElements := elem.Children("option")
	require.Equal(t, 1, len(optElements))
	optElem := optElements[0]
	require.Equal(t, "opt_label", optElem.Attribute("label"))

	valElem = optElem.Child("value")
	require.Equal(t, "An option value", valElem.Text())

	validateElem := elem.ChildNamespace("validate", validateNamespace)
	require.NotNil(t, validateElem)

	regexElem := validateElem.Child("regex")
	require.NotNil(t, regexElem)
	require.Equal(t, "([0-9]{3})-([0-9]{2})-([0-9]{4})", regexElem.Text())
}
