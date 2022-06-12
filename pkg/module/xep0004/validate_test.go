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

func TestValidator_Element(t *testing.T) {
	v := Validate{
		DataType:  StringDataType,
		Validator: &OpenValidator{},
	}

	elem := v.Element()

	require.NotNil(t, elem)
	require.Equal(t, "validate", elem.Name())
	require.Equal(t, validateNamespace, elem.Attribute(stravaganza.Namespace))
	require.Equal(t, StringDataType, elem.Attribute("datatype"))

	validatorElem := elem.Child("open")
	require.NotNil(t, validatorElem)
	require.Equal(t, "open", validatorElem.Name())
}
