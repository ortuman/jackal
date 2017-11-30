/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "fmt"

const (
	getIQType    = "get"
	setIQType    = "set"
	resultIQType = "result"
)

type IQ struct {
	Element
	to   *JID
	from *JID
}

func NewIQ(e *Element, to *JID, from *JID) (*IQ, error) {
	if e.name != "iq" {
		return nil, fmt.Errorf("wrong iq element name: %s", e.name)
	}
	if !isIQType(e.Type()) {
		return nil, fmt.Errorf("wrong iq type: %s", e.Type())
	}
	iq := &IQ{}
	iq.name = e.name
	iq.text = e.text
	iq.attrs = make([]Attribute, len(e.attrs), cap(e.attrs))
	iq.elements = make([]*Element, len(e.elements), cap(e.elements))
	copy(iq.attrs, e.attrs)
	copy(iq.elements, e.elements)
	iq.to = to
	iq.from = from
	return iq, nil
}

// IsGet returns true if this is a 'get' type IQ.
func (iq *IQ) IsGet() bool {
	return iq.Type() == getIQType
}

// IsSet returns true if this is a 'set' type IQ.
func (iq *IQ) IsSet() bool {
	return iq.Type() == setIQType
}

// IsResult returns true if this is a 'result' type IQ.
func (iq *IQ) IsResult() bool {
	return iq.Type() == resultIQType
}

// ResultIQ returns the instance associated result IQ.
func (iq *IQ) ResultIQ(from string) *IQ {
	rs := &IQ{}
	rs.name = "iq"
	rs.attrs = make([]Attribute, 0, 4)
	rs.attrs = append(rs.attrs, Attribute{"type", "result"})
	rs.attrs = append(rs.attrs, Attribute{"id", iq.ID()})
	rs.attrs = append(rs.attrs, Attribute{"to", iq.From()})
	if len(from) > 0 {
		rs.attrs = append(rs.attrs, Attribute{"from", from})
	}
	return rs
}

func isIQType(tp string) bool {
	switch tp {
	case getIQType, setIQType, resultIQType, "error":
		return true
	}
	return false
}
