// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xmpputil

import (
	"time"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/jid"
)

const delayTimeFormat = "2006-01-02T15:04:05.000Z"

// MakeResultIQ creates a new result stanza derived from iq.
func MakeResultIQ(iq *stravaganza.IQ, queryChild stravaganza.Element) *stravaganza.IQ {
	b := iq.ResultBuilder()
	if queryChild != nil {
		b.WithChild(queryChild)
	}
	resIQ, _ := b.BuildIQ()
	return resIQ
}

// MakePresence creates presence of type typ using fromJID and toJID addresses.
func MakePresence(fromJID, toJID *jid.JID, typ string, children []stravaganza.Element) *stravaganza.Presence {
	pr, _ := stravaganza.NewPresenceBuilder().
		WithAttribute(stravaganza.From, fromJID.String()).
		WithAttribute(stravaganza.To, toJID.String()).
		WithAttribute(stravaganza.Type, typ).
		WithChildren(children...).
		BuildPresence()
	return pr
}

// MakeErrorStanza creates an error stanza using errReason as reason.
func MakeErrorStanza(stanza stravaganza.Stanza, errReason stanzaerror.Reason) stravaganza.Stanza {
	errStanza, _ := stanzaerror.E(errReason, stanza).
		Stanza(false)
	return errStanza
}

// MakeErrorStanzaWithApplicationElement creates an error stanza using errReason as reason.
func MakeErrorStanzaWithApplicationElement(stanza stravaganza.Stanza, applicationElement stravaganza.Element, errReason stanzaerror.Reason) stravaganza.Stanza {
	se := stanzaerror.E(errReason, stanza)
	se.ApplicationElement = applicationElement

	errStanza, _ := se.Stanza(false)
	return errStanza
}

// MakeDelayMessage creates a new message adding delayed information.
func MakeDelayMessage(stanza stravaganza.Stanza, stamp time.Time, from, text string) *stravaganza.Message {
	sb := stravaganza.NewBuilderFromElement(stanza)
	sb.WithChild(
		stravaganza.NewBuilder("delay").
			WithAttribute(stravaganza.Namespace, "urn:xmpp:delay").
			WithAttribute(stravaganza.From, from).
			WithAttribute("stamp", stamp.UTC().Format(delayTimeFormat)).
			WithText(text).
			Build(),
	)
	dMsg, _ := sb.BuildMessage()
	return dMsg
}

// MakeStanzaIDMessage creates and returns a new message containing a stanza-id element.
func MakeStanzaIDMessage(originalMsg *stravaganza.Message, stanzaID, by string) *stravaganza.Message {
	msg, _ := stravaganza.NewBuilderFromElement(originalMsg).
		WithChild(
			stravaganza.NewBuilder("stanza-id").
				WithAttribute(stravaganza.Namespace, "urn:xmpp:sid:0").
				WithAttribute("by", by).
				WithAttribute("id", stanzaID).
				Build(),
		).
		BuildMessage()
	return msg
}

// MessageStanzaID returns the stanza-id value contained in msg parameter.
func MessageStanzaID(msg *stravaganza.Message) string {
	sidElem := msg.ChildNamespace("stanza-id", "urn:xmpp:sid:0")
	if sidElem == nil {
		return ""
	}
	return sidElem.Attribute("id")
}

// MakeForwardedStanza creates a new forwarded element derived from the passed stanza.
func MakeForwardedStanza(stanza stravaganza.Stanza, stamp *time.Time) stravaganza.Element {
	b := stravaganza.NewBuilder("forwarded").
		WithAttribute(stravaganza.Namespace, "urn:xmpp:forward:0").
		WithChild(
			stravaganza.NewBuilderFromElement(stanza).
				WithAttribute(stravaganza.Namespace, "jabber:client").
				Build(),
		)
	if stamp != nil {
		b.WithChild(
			stravaganza.NewBuilder("delay").
				WithAttribute(stravaganza.Namespace, "urn:xmpp:delay").
				WithAttribute("stamp", stamp.UTC().Format(delayTimeFormat)).
				Build(),
		)
	}
	return b.Build()
}
