/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

type MutableElement struct {
	Element
}

func NewMutableElement() *MutableElement {
	return &MutableElement{}
}

func (e *MutableElement) SetAttribute(label, value string) {
	for i := 0; i < len(e.attrs); i++ {
		if e.attrs[i].label == label {
			e.attrs[i].value = value
			return
		}
	}
	e.attrs = append(e.attrs, Attribute{label, value})
}
