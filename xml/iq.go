/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
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
	Element
	to   *JID
	from *JID
}

type MutableIQ struct {
	MutableElement
}

func NewIQ(e *Element, from *JID, to *JID) (*IQ, error) {
	if e.name != "iq" {
		return nil, fmt.Errorf("wrong IQ element name: %s", e.name)
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
	if (iqType == GetType || iqType == SetType) && len(e.elements) != 1 {
		return nil, errors.New(`an IQ stanza of type "get" or "set" must contain one and only one child element`)
	}
	if iqType == ResultType && len(e.elements) > 1 {
		return nil, errors.New(`An IQ stanza of type "result" must include zero or one child elements`)
	}
	iq := &IQ{}
	iq.name = e.name
	iq.copyAttributes(e.attrs)
	iq.copyElements(e.elements)
	iq.setAttribute("to", to.ToFullJID())
	iq.setAttribute("from", from.ToFullJID())
	iq.to = to
	iq.from = from
	return iq, nil
}

func NewMutableIQ() *MutableIQ {
	iq := &MutableIQ{}
	iq.SetName("iq")
	return iq
}

func NewMutableIQNamespace(namespace string) *MutableIQ {
	iq := &MutableIQ{}
	iq.SetName("iq")
	iq.SetNamespace(namespace)
	return iq
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
func (iq *IQ) ResultIQ() *Element {
	return iq.ResultIQFrom("")
}

// ResultIQFrom returns the instance associated result IQ
// attaching from attribute.
func (iq *IQ) ResultIQFrom(from string) *Element {
	rs := &Element{}
	rs.name = "iq"
	rs.setAttribute("type", ResultType)
	rs.setAttribute("id", iq.ID())
	rs.setAttribute("to", iq.From())
	if len(from) > 0 {
		rs.setAttribute("from", iq.From())
	}
	return rs
}

func isIQType(tp string) bool {
	switch tp {
	case GetType, SetType, ResultType, "error":
		return true
	}
	return false
}
