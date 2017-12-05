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

// IsAvailable returns true if this is an 'available' type Presence.
func (p *Presence) IsAvailable() bool {
	return p.Type() == AvailableType
}

// IsUnavailable returns true if this is an 'unavailable' type Presence.
func (p *Presence) IsUnavailable() bool {
	return p.Type() == UnavailableType
}

// IsSubscribe returns true if this is a 'subscribe' type Presence.
func (p *Presence) IsSubscribe() bool {
	return p.Type() == SubscribeType
}

// IsUnsubscribe returns true if this is an 'unsubscribe' type Presence.
func (p *Presence) IsUnsubscribe() bool {
	return p.Type() == UnsubscribeType
}

// IsSubscribed returns true if this is a 'subscribed' type Presence.
func (p *Presence) IsSubscribed() bool {
	return p.Type() == SubscribedType
}

// IsUnsubscribed returns true if this is an 'unsubscribed' type Presence.
func (p *Presence) IsUnsubscribed() bool {
	return p.Type() == UnsubscribedType
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
