// Copyright 2020 The jackal Authors
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

// MakeResultIQ creates a new result stanza derived from iq.
func MakeResultIQ(iq *stravaganza.IQ, queryChild stravaganza.Element) *stravaganza.IQ {
	b := iq.ResultBuilder()
	if queryChild != nil {
		b.WithChild(queryChild)
	}
	resIQ, _ := b.BuildIQ(false)
	return resIQ
}

// MakePresence creates presence of type typ using fromJID and toJID addresses.
func MakePresence(fromJID, toJID *jid.JID, typ string, children []stravaganza.Element) *stravaganza.Presence {
	pr, _ := stravaganza.NewPresenceBuilder().
		WithAttribute(stravaganza.From, fromJID.String()).
		WithAttribute(stravaganza.To, toJID.String()).
		WithAttribute(stravaganza.Type, typ).
		WithChildren(children...).
		BuildPresence(false)
	return pr
}

// MakeErrorStanza creates an error stanza using errReason as reason.
func MakeErrorStanza(stanza stravaganza.Stanza, errReason stanzaerror.Reason) stravaganza.Stanza {
	errStanza, _ := stanzaerror.E(errReason, stanza).
		Stanza(false)
	return errStanza
}

// MakeDelayMessage creates a new message adding delayed information.
func MakeDelayMessage(stanza stravaganza.Stanza, stamp time.Time, from, text string) *stravaganza.Message {
	sb := stravaganza.NewBuilderFromElement(stanza)
	sb.WithChild(
		stravaganza.NewBuilder("delay").
			WithAttribute(stravaganza.Namespace, "urn:xmpp:delay").
			WithAttribute(stravaganza.From, from).
			WithAttribute("stamp", stamp.UTC().Format(time.RFC3339)).
			WithText(text).
			Build(),
	)
	dMsg, _ := sb.BuildMessage(false)
	return dMsg
}
