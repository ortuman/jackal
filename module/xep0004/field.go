/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0004

import (
	"fmt"

	"github.com/ortuman/jackal/xmpp"
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

// NewFieldFromElement returns a new form field entity reading it
// from it's XMPP representation.
func NewFieldFromElement(elem xmpp.XElement) (*Field, error) {
	if n := elem.Name(); n != "field" {
		return nil, fmt.Errorf("invalid field name: %s", n)
	}
	v := elem.Attributes().Get("var")

	typ := elem.Attributes().Get("type")
	if len(typ) > 0 && !isValidFieldType(typ) {
		return nil, fmt.Errorf("invalid field type: %s", typ)
	}
	label := elem.Attributes().Get("label")

	f := &Field{
		Var:   v,
		Type:  typ,
		Label: label,
	}
	if desc := elem.Elements().Child("desc"); desc != nil {
		f.Description = desc.Text()
	}
	if required := elem.Elements().Child("required"); required != nil {
		f.Required = true
	}
	values := elem.Elements().Children("value")
	for _, val := range values {
		f.Values = append(f.Values, val.Text())
	}
	options := elem.Elements().Children("option")
	for _, opt := range options {
		var label, value string
		label = opt.Attributes().Get("label")
		if v := opt.Elements().Child("value"); v != nil {
			value = v.Text()
		}
		f.Options = append(f.Options, Option{Label: label, Value: value})
	}
	return f, nil
}

// Element returns form field XMPP representation.
func (f *Field) Element() xmpp.XElement {
	el := xmpp.NewElementName("field")
	if len(f.Type) > 0 {
		el.SetAttribute("type", f.Type)
	}
	if len(f.Label) > 0 {
		el.SetAttribute("label", f.Label)
	}
	if len(f.Var) > 0 {
		el.SetAttribute("var", f.Var)
	}
	if f.Required {
		el.AppendElement(xmpp.NewElementName("required"))
	}
	if len(f.Description) > 0 {
		descEl := xmpp.NewElementName("desc")
		descEl.SetText(f.Description)
		el.AppendElement(descEl)
	}
	for _, value := range f.Values {
		valEl := xmpp.NewElementName("value")
		valEl.SetText(value)
		el.AppendElement(valEl)
	}
	for _, option := range f.Options {
		optEl := xmpp.NewElementName("option")
		if len(option.Label) > 0 {
			optEl.SetAttribute("label", option.Label)
		}
		valEl := xmpp.NewElementName("value")
		valEl.SetText(option.Value)
		optEl.AppendElement(valEl)
		el.AppendElement(optEl)
	}
	return el
}

func isValidFieldType(typ string) bool {
	switch typ {
	case Boolean, Fixed, Hidden, JidMulti, JidSingle, ListMulti,
		ListSingle, TextMulti, TextPrivate, TextSingle:
		return true
	}
	return false
}
