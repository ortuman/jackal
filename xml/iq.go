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
	if (iqType == getIQType || iqType == setIQType) && len(e.elements) != 1 {
		return nil, errors.New(`an IQ stanza of type "get" or "set" must contain one and only one child element`)
	}
	if iqType == resultIQType && len(e.elements) > 1 {
		return nil, errors.New(`An IQ stanza of type "result" must include zero or one child elements`)
	}
	iq := &IQ{}
	iq.name = e.name
	iq.copyAttributes(e.attrs)
	iq.copyElements(e.elements)
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
	rs.setAttribute("type", resultIQType)
	rs.setAttribute("id", iq.ID())
	rs.setAttribute("to", iq.From())
	if len(from) > 0 {
		rs.setAttribute("from", iq.From())
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
