/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "fmt"

const (
	AvailableType    = ""
	UnavailableType  = "unavailable"
	SubscribeType    = "subscribe"
	UnsubscribeType  = "unsubscribe"
	SubscribedType   = "subscribed"
	UnsubscribedType = "unsubscribed"
)

type Presence struct {
	Element
	to   *JID
	from *JID
}

type MutablePresence struct {
	MutableElement
}

func NewPresence(e *Element, from *JID, to *JID) (*Presence, error) {
	if e.name != "presence" {
		return nil, fmt.Errorf("wrong Presence element name: %s", e.name)
	}
	presenceType := e.Type()
	if !isPresenceType(presenceType) {
		return nil, fmt.Errorf(`invalid Presence "type" attribute: %s`, presenceType)
	}
	p := &Presence{}
	p.name = e.name
	p.copyAttributes(e.attrs)
	p.copyElements(e.elements)
	p.setAttribute("to", to.ToFullJID())
	p.setAttribute("from", from.ToFullJID())
	p.to = to
	p.from = from
	return p, nil
}

func NewMutablePresence() *MutablePresence {
	p := &MutablePresence{}
	p.SetName("presence")
	return p
}

func NewMutablePresenceType(presenceType string) *MutablePresence {
	p := &MutablePresence{}
	p.SetName("presence")
	p.SetType(presenceType)
	return p
}

// ToJID satisfies stanza interface.
func (p *Presence) ToJID() *JID {
	return p.to
}

// FromJID satisfies stanza interface.
func (p *Presence) FromJID() *JID {
	return p.from
}

func isPresenceType(presenceType string) bool {
	switch presenceType {
	case AvailableType, UnavailableType, SubscribeType, UnsubscribeType, SubscribedType, UnsubscribedType:
		return true
	default:
		return false
	}
}
