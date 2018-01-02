/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package entity

import (
	"errors"
	"fmt"

	"github.com/ortuman/jackal/xml"
)

type RosterItem struct {
	JID          *xml.JID
	Name         string
	Subscription string
	Ask          bool
	Groups       []string
}

func NewRosterItem(item *xml.Element) (*RosterItem, error) {
	ri := &RosterItem{}
	if jid := item.Attribute("jid"); len(jid) > 0 {
		j, err := xml.NewJIDString(jid, false)
		if err != nil {
			return nil, err
		}
		ri.JID = j
	} else {
		return nil, errors.New("item 'jid' attribute is required")
	}
	ri.Name = item.Attribute("name")

	subscription := item.Attribute("subscription")
	if len(subscription) > 0 {
		switch subscription {
		case "both", "from", "none", "remove", "to":
			break
		default:
			return nil, fmt.Errorf("unrecognized 'subscription' enum type: %s", subscription)
		}
		ri.Subscription = subscription
	} else {
		ri.Subscription = "none"
	}
	ask := item.Attribute("ask")
	if len(ask) > 0 {
		if ask != "subscribe" {
			return nil, fmt.Errorf("unrecognized 'ask' enum type: %s", subscription)
		}
		ri.Ask = true
	}
	groups := item.FindElements("group")
	for _, group := range groups {
		if group.AttributesCount() > 0 {
			return nil, errors.New("group element must not contain any attribute")
		}
		ri.Groups = append(ri.Groups, group.Text())
	}
	return ri, nil
}

func (ri *RosterItem) Element() *xml.Element {
	item := xml.NewMutableElementName("item")
	item.SetAttribute("jid", ri.JID.ToBareJID())
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
		gr := xml.NewMutableElementName("group")
		gr.SetText(group)
		item.AppendElement(gr.Copy())
	}
	return item.Copy()
}
