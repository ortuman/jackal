/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"encoding/gob"
	"errors"
	"fmt"
	"strconv"

	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	// AvailableType represents an 'available' Presence type.
	AvailableType = ""

	// UnavailableType represents a 'unavailable' Presence type.
	UnavailableType = "unavailable"

	// SubscribeType represents a 'subscribe' Presence type.
	SubscribeType = "subscribe"

	// UnsubscribeType represents a 'unsubscribe' Presence type.
	UnsubscribeType = "unsubscribe"

	// SubscribedType represents a 'subscribed' Presence type.
	SubscribedType = "subscribed"

	// UnsubscribedType represents a 'unsubscribed' Presence type.
	UnsubscribedType = "unsubscribed"

	// ProbeType represents a 'probe' Presence type.
	ProbeType = "probe"
)

// ShowState represents Presence show state.
type ShowState int

const (
	// AvailableShowState represents 'available' Presence show state.
	AvailableShowState ShowState = iota

	// AwayShowState represents 'away' Presence show state.
	AwayShowState

	// ChatShowState represents 'chat' Presence show state.
	ChatShowState

	// DoNotDisturbShowState represents 'dnd' Presence show state.
	DoNotDisturbShowState

	// ExtendedAwaysShowState represents 'xa' Presence show state.
	ExtendedAwaysShowState
)

// Presence type represents an <presence> element.
// All incoming <presence> elements providing from the
// stream will automatically be converted to Presence objects.
type Presence struct {
	stanzaElement
	showState ShowState
	priority  int8
}

// NewPresenceFromElement creates a Presence object from XElement.
func NewPresenceFromElement(e XElement, from *jid.JID, to *jid.JID) (*Presence, error) {
	if e.Name() != "presence" {
		return nil, fmt.Errorf("wrong Presence element name: %s", e.Name())
	}
	presenceType := e.Type()
	if !isPresenceType(presenceType) {
		return nil, fmt.Errorf(`invalid Presence "type" attribute: %s`, presenceType)
	}
	p := &Presence{}
	p.copyFrom(e)

	// show
	if err := p.setShow(); err != nil {
		return nil, err
	}
	// status
	if err := p.validateStatus(); err != nil {
		return nil, err
	}
	// priority
	if err := p.setPriority(); err != nil {
		return nil, err
	}
	p.SetFromJID(from)
	p.SetToJID(to)
	p.SetNamespace("")
	return p, nil
}

// NewPresence creates and returns a new Presence element.
func NewPresence(from *jid.JID, to *jid.JID, presenceType string) *Presence {
	p := &Presence{}
	p.SetName("presence")
	p.SetFromJID(from)
	p.SetToJID(to)
	p.SetType(presenceType)
	return p
}

// NewPresenceFromGob creates and returns a new Presence element from a given gob decoder.
func NewPresenceFromGob(dec *gob.Decoder) (*Presence, error) {
	p := &Presence{}
	if err := p.FromGob(dec); err != nil {
		return nil, err
	}
	return p, nil
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

// IsProbe returns true if this is an 'probe' type Presence.
func (p *Presence) IsProbe() bool {
	return p.Type() == ProbeType
}

// Status returns presence stanza default status.
func (p *Presence) Status() string {
	if st := p.Elements().Child("status"); st != nil {
		return st.Text()
	}
	return ""
}

// ShowState returns presence stanza show state.
func (p *Presence) ShowState() ShowState {
	return p.showState
}

// Priority returns presence stanza priority value.
func (p *Presence) Priority() int8 {
	return p.priority
}

// FromGob deserializes an element node from it's gob binary representation.
func (p *Presence) FromGob(dec *gob.Decoder) error {
	dec.Decode(&p.name)
	dec.Decode(&p.text)
	p.attrs.fromGob(dec)
	p.elements.fromGob(dec)

	// set from and to JIDs
	fromJID, err := jid.NewWithString(p.From(), false)
	if err != nil {
		return err
	}
	toJID, err := jid.NewWithString(p.To(), false)
	if err != nil {
		return err
	}
	p.SetFromJID(fromJID)
	p.SetToJID(toJID)
	return nil
}

func isPresenceType(presenceType string) bool {
	switch presenceType {
	case ErrorType, AvailableType, UnavailableType, SubscribeType,
		UnsubscribeType, SubscribedType, UnsubscribedType, ProbeType:
		return true
	default:
		return false
	}
}

func (p *Presence) validateStatus() error {
	sts := p.elements.Children("status")
	for _, st := range sts {
		switch st.Attributes().Count() {
		case 0:
			break
		case 1:
			as := st.Attributes()
			if as.(attributeSet)[0].Label == "xml:lang" {
				break
			}
			fallthrough
		default:
			return errors.New(" the <status/> element MUST NOT possess any attributes, with the exception of the 'xml:lang' attribute")
		}
	}
	return nil
}

func (p *Presence) setShow() error {
	shs := p.elements.Children("show")
	switch len(shs) {
	case 0:
		p.showState = AvailableShowState
	case 1:
		if shs[0].Attributes().Count() > 0 {
			return errors.New(" the <show/> element MUST NOT possess any attributes")
		}
		switch shs[0].Text() {
		case "away":
			p.showState = AwayShowState
		case "chat":
			p.showState = ChatShowState
		case "dnd":
			p.showState = DoNotDisturbShowState
		case "xa":
			p.showState = ExtendedAwaysShowState
		default:
			return fmt.Errorf("invalid Presence show state: %s", shs[0].Text())
		}

	default:
		return errors.New(" Presence stanza MUST NOT contain more than one <show/> element")
	}
	return nil
}

func (p *Presence) setPriority() error {
	ps := p.elements.Children("priority")
	switch len(ps) {
	case 0:
		break
	case 1:
		pr, err := strconv.Atoi(ps[0].Text())
		if err != nil {
			return err
		}
		if pr < -128 || pr > 127 {
			return errors.New("priority value MUST be an integer between -128 and +127")
		}
		p.priority = int8(pr)

	default:
		return errors.New("a Presence stanza MUST NOT contain more than one <priority/> element")
	}
	return nil
}
