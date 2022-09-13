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

// Fields represent a set of form fields
type Fields []Field

// ValueForField returns the associated value for a given field name.
func (f Fields) ValueForField(fieldName string) string {
	return f.ValueForFieldOfType(fieldName, "")
}

// ValuesForField returns all associated values for a given field name.
func (f Fields) ValuesForField(fieldName string) []string {
	return f.ValuesForFieldOfType(fieldName, "")
}

// ValueForFieldOfType returns the associated value for a given field name and type.
func (f Fields) ValueForFieldOfType(fieldName, typ string) string {
	for _, field := range f {
		if field.Var == fieldName && field.Type == typ && len(field.Values) > 0 {
			return field.Values[0]
		}
	}
	return ""
}

// ValuesForFieldOfType returns all associated values for a given field name and type.
func (f Fields) ValuesForFieldOfType(fieldName, typ string) []string {
	var res []string
	for _, field := range f {
		if field.Var == fieldName && field.Type == typ && len(field.Values) > 0 {
			res = append(res, field.Values...)
		}
	}
	return res
}
