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

// FormNamespace specifies XEP-0004 namespace constant value.
const FormNamespace = "jabber:x:data"

const (
	// Form represents a 'form' data form.
	Form = "form"

	// Submit represents a 'submit' data form.
	Submit = "submit"

	// Cancel represents a 'cancel' data form.
	Cancel = "cancel"

	// Result represents a 'result' data form.
	Result = "result"
)

// DataForm represents a form that could be use for gathering data
// as well as for reporting data returned from a search.
type DataForm struct {
	Type         string
	Title        string
	Instructions string
	Fields       Fields
	Reported     Fields
	Items        []Fields
}

// NewFormFromElement returns a new data form entity reading it from it's XMPP representation.
func NewFormFromElement(elem stravaganza.Element) (*DataForm, error) {
	if n := elem.Name(); n != "x" {
		return nil, fmt.Errorf("xep0004: invalid form name: %s", n)
	}
	if ns := elem.Attribute(stravaganza.Namespace); ns != FormNamespace {
		return nil, fmt.Errorf("xep0004: invalid form namespace: %s", ns)
	}
	typ := elem.Attribute("type")
	if !isValidFormType(typ) {
		return nil, fmt.Errorf("xep0004: invalid form type: %s", typ)
	}
	f := &DataForm{Type: typ}

	if title := elem.Child("title"); title != nil {
		f.Title = title.Text()
	}
	if inst := elem.Child("instructions"); inst != nil {
		f.Instructions = inst.Text()
	}
	if reportedElem := elem.Child("reported"); reportedElem != nil {
		fields, err := fieldsFromElement(reportedElem)
		if err != nil {
			return nil, err
		}
		f.Reported = fields
	}
	items := elem.Children("item")
	for _, item := range items {
		fields, err := fieldsFromElement(item)
		if err != nil {
			return nil, err
		}
		f.Items = append(f.Items, fields)
	}
	fields, err := fieldsFromElement(elem)
	if err != nil {
		return nil, err
	}
	f.Fields = fields
	return f, nil
}

// Element returns data form XML representation.
func (f *DataForm) Element() stravaganza.Element {
	sb := stravaganza.NewBuilder("x")
	sb.WithAttribute(stravaganza.Namespace, FormNamespace)
	if len(f.Title) > 0 {
		sb.WithChild(
			stravaganza.NewBuilder("title").
				WithText(f.Title).
				Build(),
		)
	}
	if len(f.Type) > 0 {
		sb.WithAttribute("type", f.Type)
	}
	if len(f.Instructions) > 0 {
		sb.WithChild(
			stravaganza.NewBuilder("instructions").
				WithText(f.Instructions).
				Build(),
		)
	}
	if len(f.Reported) > 0 {
		reportedBuilder := stravaganza.NewBuilder("reported")
		for _, field := range f.Reported {
			reportedBuilder.WithChild(field.Element())
		}
		sb.WithChild(reportedBuilder.Build())
	}
	if len(f.Items) > 0 {
		for _, item := range f.Items {
			itemBuilder := stravaganza.NewBuilder("item")
			for _, field := range item {
				itemBuilder.WithChild(field.Element())
			}
			sb.WithChild(itemBuilder.Build())
		}
	}
	for _, field := range f.Fields {
		sb.WithChild(field.Element())
	}
	return sb.Build()
}

func fieldsFromElement(elem stravaganza.Element) ([]Field, error) {
	var res []Field
	fields := elem.Children("field")
	for _, fieldElem := range fields {
		field, err := NewFieldFromElement(fieldElem)
		if err != nil {
			return nil, err
		}
		res = append(res, *field)
	}
	return res, nil
}

func isValidFormType(typ string) bool {
	switch typ {
	case Form, Submit, Cancel, Result:
		return true
	}
	return false
}
