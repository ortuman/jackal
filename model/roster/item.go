/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"bytes"
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

// FromBytes deserializes a RosterItem entity from its representation.
func (ri *Item) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&ri.Username); err != nil {
		return err
	}
	if err := dec.Decode(&ri.JID); err != nil {
		return err
	}
	if err := dec.Decode(&ri.Name); err != nil {
		return err
	}
	if err := dec.Decode(&ri.Subscription); err != nil {
		return err
	}
	if err := dec.Decode(&ri.Ask); err != nil {
		return err
	}
	if err := dec.Decode(&ri.Ver); err != nil {
		return err
	}
	return dec.Decode(&ri.Groups)
}

// ToBytes converts a RosterItem entity to its binary representation.
func (ri *Item) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&ri.Username); err != nil {
		return err
	}
	if err := enc.Encode(&ri.JID); err != nil {
		return err
	}
	if err := enc.Encode(&ri.Name); err != nil {
		return err
	}
	if err := enc.Encode(&ri.Subscription); err != nil {
		return err
	}
	if err := enc.Encode(&ri.Ask); err != nil {
		return err
	}
	if err := enc.Encode(&ri.Ver); err != nil {
		return err
	}
	return enc.Encode(&ri.Groups)
}
