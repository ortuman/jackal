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
	// GetType represents a 'get' IQ type.
	GetType = "get"

	// SetType represents a 'set' IQ type.
	SetType = "set"

	// ResultType represents a 'result' IQ type.
	ResultType = "result"
)

// IQ type represents an <iq> element.
// All incoming <iq> elements providing from the
// stream will automatically be converted to IQ objects.
type IQ struct {
	Element
	to   *JID
	from *JID
}

// NewIQFromElement creates an IQ object from ElementNode.
func NewIQFromElement(e ElementNode, from *JID, to *JID) (*IQ, error) {
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
	if (iqType == GetType || iqType == SetType) && e.Elements().Count() != 1 {
		return nil, errors.New(`an IQ stanza of type "get" or "set" must contain one and only one child element`)
	}
	if iqType == ResultType && e.Elements().Count() > 1 {
		return nil, errors.New(`an IQ stanza of type "result" must include zero or one child elements`)
	}
	iq := &IQ{}
	iq.copyFrom(e)
	iq.SetToJID(to)
	iq.SetFromJID(from)
	return iq, nil
}

// NewIQType creates and returns a new IQ element.
func NewIQType(identifier string, iqType string) *IQ {
	iq := &IQ{}
	iq.SetName("iq")
	iq.SetID(identifier)
	iq.SetType(iqType)
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
func (iq *IQ) ResultIQ() *IQ {
	rs := &IQ{}
	rs.SetName("iq")
	rs.SetAttribute("xmlns", iq.Namespace())
	rs.SetAttribute("type", ResultType)
	rs.SetAttribute("id", iq.ID())
	rs.SetAttribute("from", iq.To())
	rs.SetAttribute("to", iq.From())
	return rs
}

// ToJID returns iq 'from' JID value.
func (iq *IQ) ToJID() *JID {
	return iq.to
}

// SetToJID sets the IQ 'to' JID value.
func (iq *IQ) SetToJID(to *JID) {
	iq.to = to
	iq.SetAttribute("to", to.String())
}

// FromJID returns presence 'from' JID value.
func (iq *IQ) FromJID() *JID {
	return iq.from
}

// SetFromJID sets the IQ 'from' JID value.
func (iq *IQ) SetFromJID(from *JID) {
	iq.from = from
	iq.SetAttribute("from", from.String())
}

func isIQType(tp string) bool {
	switch tp {
	case ErrorType, GetType, SetType, ResultType:
		return true
	}
	return false
}
