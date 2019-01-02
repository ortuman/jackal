/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// roster item subscription values
const (
	SubscriptionNone   = "none"
	SubscriptionFrom   = "from"
	SubscriptionTo     = "to"
	SubscriptionBoth   = "both"
	SubscriptionRemove = "remove"
)

// Item represents a roster item storage entity.
type Item struct {
	Username     string
	JID          string
	Name         string
	Subscription string
	Ask          bool
	Ver          int
	Groups       []string
}

// NewItem parses an XML element returning a derived roster item instance.
func NewItem(elem xmpp.XElement) (*Item, error) {
	if elem.Name() != "item" {
		return nil, fmt.Errorf("invalid item element name: %s", elem.Name())
	}
	ri := &Item{}
	if jidStr := elem.Attributes().Get("jid"); len(jidStr) > 0 {
		j, err := jid.NewWithString(jidStr, false)
		if err != nil {
			return nil, err
		}
		ri.JID = j.String()
	} else {
		return nil, errors.New("item 'jid' attribute is required")
	}
	ri.Name = elem.Attributes().Get("name")

	subscription := elem.Attributes().Get("subscription")
	if len(subscription) > 0 {
		switch subscription {
		case SubscriptionBoth, SubscriptionFrom, SubscriptionTo, SubscriptionNone, SubscriptionRemove:
			break
		default:
			return nil, fmt.Errorf("unrecognized 'subscription' enum type: %s", subscription)
		}
		ri.Subscription = subscription
	}
	ask := elem.Attributes().Get("ask")
	if len(ask) > 0 {
		if ask != "subscribe" {
			return nil, fmt.Errorf("unrecognized 'ask' enum type: %s", subscription)
		}
		ri.Ask = true
	}
	groups := elem.Elements().Children("group")
	for _, group := range groups {
		if group.Attributes().Count() > 0 {
			return nil, errors.New("group element must not contain any attribute")
		}
		if len(group.Text()) > 0 {
			ri.Groups = append(ri.Groups, group.Text())
		}
	}
	return ri, nil
}

// Element returns a roster item XML element representation.
func (ri *Item) Element() xmpp.XElement {
	riJID := ri.ContactJID()
	item := xmpp.NewElementName("item")
	item.SetAttribute("jid", riJID.ToBareJID().String())
	if len(ri.Name) > 0 {
		item.SetAttribute("name", ri.Name)
	}
	if len(ri.Subscription) > 0 {
		item.SetAttribute("subscription", ri.Subscription)
	}
	if ri.Ask {
		item.SetAttribute("ask", "subscribe")
	}
	for _, group := range ri.Groups {
		gr := xmpp.NewElementName("group")
		gr.SetText(group)
		item.AppendElement(gr)
	}
	return item
}

// ContactJID parses and returns roster item contact JID.
func (ri *Item) ContactJID() *jid.JID {
	j, _ := jid.NewWithString(ri.JID, true)
	return j
}

// FromGob deserializes a RosterItem entity from it's gob binary representation.
func (ri *Item) FromGob(dec *gob.Decoder) error {
	dec.Decode(&ri.Username)
	dec.Decode(&ri.JID)
	dec.Decode(&ri.Name)
	dec.Decode(&ri.Subscription)
	dec.Decode(&ri.Ask)
	dec.Decode(&ri.Ver)
	dec.Decode(&ri.Groups)
	return nil
}

// ToGob converts a RosterItem entity
// to it's gob binary representation.
func (ri *Item) ToGob(enc *gob.Encoder) {
	enc.Encode(&ri.Username)
	enc.Encode(&ri.JID)
	enc.Encode(&ri.Name)
	enc.Encode(&ri.Subscription)
	enc.Encode(&ri.Ask)
	enc.Encode(&ri.Ver)
	enc.Encode(&ri.Groups)
}
