/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"errors"
	"fmt"
)

const (
	GetType    = "get"
	SetType    = "set"
	ResultType = "result"
)

type IQ struct {
	XElement
	to   *JID
	from *JID
}

func NewIQType(identifier string, iqType string) *IQ {
	iq := &IQ{}
	iq.SetName("iq")
	iq.SetID(identifier)
	iq.SetType(iqType)
	return iq
}

func NewIQFromElement(e Element, from *JID, to *JID) (*IQ, error) {
	if e.Name() != "iq" {
		return nil, fmt.Errorf("wrong IQ element name: %s", e.Name())
	}
	if len(e.ID()) == 0 {
		return nil, errors.New(`IQ "id" attribute is required`)
	}
	iqType := e.Type()
	if len(iqType) == 0 {
		return nil, errors.New(`IQ "type" attribute is required`)
	}
	if !isIQType(iqType) {
		return nil, fmt.Errorf(`invalid IQ "type" attribute: %s`, iqType)
	}
	if (iqType == GetType || iqType == SetType) && len(e.Elements()) != 1 {
		return nil, errors.New(`an IQ stanza of type "get" or "set" must contain one and only one child element`)
	}
	if iqType == ResultType && len(e.Elements()) > 1 {
		return nil, errors.New(`An IQ stanza of type "result" must include zero or one child elements`)
	}
	iq := &IQ{}
	iq.name = e.Name()
	iq.attrs = e.Attributes()
	iq.elements = e.Elements()
	iq.SetAttribute("to", to.ToFullJID())
	iq.SetAttribute("from", from.ToFullJID())
	iq.to = to
	iq.from = from
	return iq, nil
}

// IsGet returns true if this is a 'get' type IQ.
func (iq *IQ) IsGet() bool {
	return iq.Type() == GetType
}

// IsSet returns true if this is a 'set' type IQ.
func (iq *IQ) IsSet() bool {
	return iq.Type() == SetType
}

// IsResult returns true if this is a 'result' type IQ.
func (iq *IQ) IsResult() bool {
	return iq.Type() == ResultType
}

// ResultIQ returns the instance associated result IQ.
func (iq *IQ) ResultIQ() *IQ {
	rs := &IQ{}
	rs.SetName("iq")
	rs.SetAttribute("type", ResultType)
	rs.SetAttribute("id", iq.ID())
	rs.SetAttribute("from", iq.To())
	rs.SetAttribute("to", iq.From())
	return rs
}

// ToJID satisfies stanza interface.
func (iq *IQ) ToJID() *JID {
	return iq.to
}

// FromJID satisfies stanza interface.
func (iq *IQ) FromJID() *JID {
	return iq.from
}

// Copy returns a deep copy of this message stanza.
func (iq *IQ) Copy() *IQ {
	cp := &IQ{}
	cp.name = iq.name
	cp.text = iq.text
	cp.attrs = iq.attrs
	cp.elements = iq.elements
	return cp
}

func isIQType(tp string) bool {
	switch tp {
	case GetType, SetType, ResultType, "error":
		return true
	}
	return false
}
