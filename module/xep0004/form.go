/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0004

import (
	"fmt"

	"github.com/ortuman/jackal/xmpp"
)

const FormNamespace = "jabber:x:data"

const (
	// SubmitForm represents a 'form' data form.
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

// NewFormFromElement returns a new data form entity reading it
// from it's XMPP representation.
func NewFormFromElement(elem xmpp.XElement) (*DataForm, error) {
	if n := elem.Name(); n != "x" {
		return nil, fmt.Errorf("invalid form name: %s", n)
	}
	if ns := elem.Namespace(); ns != FormNamespace {
		return nil, fmt.Errorf("invalid form namespace: %s", ns)
	}
	typ := elem.Attributes().Get("type")
	if !isValidFormType(typ) {
		return nil, fmt.Errorf("invalid form type: %s", typ)
	}
	f := &DataForm{Type: typ}

	if title := elem.Elements().Child("title"); title != nil {
		f.Title = title.Text()
	}
	if inst := elem.Elements().Child("instructions"); inst != nil {
		f.Instructions = inst.Text()
	}
	if reportedElem := elem.Elements().Child("reported"); reportedElem != nil {
		fields, err := fieldsFromElement(reportedElem)
		if err != nil {
			return nil, err
		}
		f.Reported = fields
	}
	itemElems := elem.Elements().Children("item")
	for _, itemElem := range itemElems {
		fields, err := fieldsFromElement(itemElem)
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

// Element returns data form XMPP representation.
func (f *DataForm) Element() xmpp.XElement {
	elem := xmpp.NewElementNamespace("x", FormNamespace)
	if len(f.Title) > 0 {
		titleElem := xmpp.NewElementName("title")
		titleElem.SetText(f.Title)
		elem.AppendElement(titleElem)
	}
	if len(f.Type) > 0 {
		elem.SetAttribute("type", f.Type)
	}
	if len(f.Instructions) > 0 {
		instElem := xmpp.NewElementName("instructions")
		instElem.SetText(f.Instructions)
		elem.AppendElement(instElem)
	}
	if len(f.Reported) > 0 {
		reportedElem := xmpp.NewElementName("reported")
		for _, field := range f.Reported {
			reportedElem.AppendElement(field.Element())
		}
		elem.AppendElement(reportedElem)
	}
	if len(f.Items) > 0 {
		for _, item := range f.Items {
			itemElem := xmpp.NewElementName("item")
			for _, field := range item {
				itemElem.AppendElement(field.Element())
			}
			elem.AppendElement(itemElem)
		}
	}
	for _, field := range f.Fields {
		elem.AppendElement(field.Element())
	}
	return elem
}

func fieldsFromElement(elem xmpp.XElement) ([]Field, error) {
	var res []Field
	fields := elem.Elements().Children("field")
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
