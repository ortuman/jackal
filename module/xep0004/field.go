// Copyright 2020 The jackal Authors
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
	"fmt"

	"github.com/jackal-xmpp/stravaganza"
)

// FormType represents form type constant value.
const FormType = "FORM_TYPE"

const (
	// Boolean represents a 'boolean' form field.
	Boolean = "boolean"

	// Fixed represents a 'fixed' form field.
	Fixed = "fixed"

	// Hidden represents a 'hidden' form field.
	Hidden = "hidden"

	// JidMulti represents a 'jid-multi' form field.
	JidMulti = "jid-multi"

	// JidSingle represents a 'jid-single' form field.
	JidSingle = "jid-single"

	// ListMulti represents a 'list-multi' form field.
	ListMulti = "list-multi"

	// ListSingle represents a 'list-single' form field.
	ListSingle = "list-single"

	// TextMulti represents a 'text-multi' form field.
	TextMulti = "text-multi"

	// TextPrivate represents a 'text-private' form field.
	TextPrivate = "text-private"

	// TextSingle represents a 'text-single' form field.
	TextSingle = "text-single"
)

// Option represents an individual field option.
type Option struct {
	Label string
	Value string
}

// Field represents a field of a form. The field could be used to represent a question to complete,
// a completed question or a data returned from a search.
type Field struct {
	Var         string
	Required    bool
	Type        string
	Label       string
	Description string
	Values      []string
	Options     []Option
}

// NewFieldFromElement returns a new form field entity reading it from it's XML representation.
func NewFieldFromElement(elem stravaganza.Element) (*Field, error) {
	if n := elem.Name(); n != "field" {
		return nil, fmt.Errorf("xep0004: invalid field name: %s", n)
	}
	v := elem.Attribute("var")

	typ := elem.Attribute("type")
	if len(typ) > 0 && !isValidFieldType(typ) {
		return nil, fmt.Errorf("xep0004: invalid field type: %s", typ)
	}
	label := elem.Attribute("label")

	f := &Field{
		Var:   v,
		Type:  typ,
		Label: label,
	}
	if desc := elem.Child("desc"); desc != nil {
		f.Description = desc.Text()
	}
	if required := elem.Child("required"); required != nil {
		f.Required = true
	}
	values := elem.Children("value")
	for _, val := range values {
		f.Values = append(f.Values, val.Text())
	}
	options := elem.Children("option")
	for _, opt := range options {
		var label, value string
		label = opt.Attribute("label")
		if v := opt.Child("value"); v != nil {
			value = v.Text()
		}
		f.Options = append(f.Options, Option{Label: label, Value: value})
	}
	return f, nil
}

// Element returns form field XML representation.
func (f *Field) Element() stravaganza.Element {
	b := stravaganza.NewBuilder("field")
	if len(f.Type) > 0 {
		b.WithAttribute("type", f.Type)
	}
	if len(f.Label) > 0 {
		b.WithAttribute("label", f.Label)
	}
	if len(f.Var) > 0 {
		b.WithAttribute("var", f.Var)
	}
	if f.Required {
		b.WithChild(stravaganza.NewBuilder("required").Build())
	}
	if len(f.Description) > 0 {
		b.WithChild(
			stravaganza.NewBuilder("desc").
				WithText(f.Description).
				Build(),
		)
	}
	for _, value := range f.Values {
		b.WithChild(
			stravaganza.NewBuilder("value").
				WithText(value).
				Build(),
		)
	}
	for _, option := range f.Options {
		sb := stravaganza.NewBuilder("option")
		if len(option.Label) > 0 {
			sb.WithAttribute("label", option.Label)
		}
		sb.WithChild(
			stravaganza.NewBuilder("value").
				WithText(option.Value).
				Build(),
		)
		b.WithChild(sb.Build())
	}
	return b.Build()
}

func isValidFieldType(typ string) bool {
	switch typ {
	case Boolean, Fixed, Hidden, JidMulti, JidSingle, ListMulti,
		ListSingle, TextMulti, TextPrivate, TextSingle:
		return true
	}
	return false
}
