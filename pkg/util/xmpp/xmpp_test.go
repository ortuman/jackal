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
	"testing"
	"time"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/stretchr/testify/require"
)

func TestMakePresence(t *testing.T) {
	// given
	from, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	to, _ := jid.NewWithString("noelia@jackal.im/balcony", true)

	// when
	children := []stravaganza.Element{
		stravaganza.NewBuilder("show").
			WithText("away").
			Build(),
	}
	p := MakePresence(from, to, stravaganza.ProbeType, children)

	// then
	require.NotNil(t, p)

	require.Equal(t, from.String(), p.FromJID().String())
	require.Equal(t, to.String(), p.ToJID().String())
	require.Equal(t, stravaganza.ProbeType, p.Attribute("type"))
	require.Len(t, p.AllChildren(), 1)
}

func TestMakeResultIQ(t *testing.T) {
	// given
	from, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	to, _ := jid.NewWithString("noelia@jackal.im/balcony", true)

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "iq1234").
		WithAttribute(stravaganza.From, from.String()).
		WithAttribute(stravaganza.To, to.String()).
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithChild(
			stravaganza.NewBuilder("ping").
				WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
				Build(),
		).
		BuildIQ()

	// when
	resIQ := MakeResultIQ(iq, stravaganza.NewBuilder("ping").
		WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
		Build(),
	)

	// then
	require.NotNil(t, resIQ)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))
	require.Equal(t, "iq1234", resIQ.Attribute("id"))
	require.Len(t, iq.AllChildren(), 1)
}

func TestMakeErrorStanza(t *testing.T) {
	// given
	from, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	to, _ := jid.NewWithString("noelia@jackal.im/balcony", true)

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "iq1234").
		WithAttribute(stravaganza.From, from.String()).
		WithAttribute(stravaganza.To, to.String()).
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithChild(
			stravaganza.NewBuilder("ping").
				WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
				Build(),
		).
		BuildIQ()

	// when
	errStanza := MakeErrorStanza(iq, stanzaerror.BadRequest)

	// then
	errEl := errStanza.Child("error")
	require.NotNil(t, errEl)
}

func TestMakeDelayStanza(t *testing.T) {
	// given
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	stamp, _ := time.Parse(time.RFC3339, "2021-02-15T15:00:00Z")
	dMsg := MakeDelayMessage(msg, stamp, "jackal.im", "Delayed IQ")

	// then
	dChild := dMsg.Child("delay")
	require.NotNil(t, dChild)
	require.Equal(t, "jackal.im", dChild.Attribute(stravaganza.From))
	require.Equal(t, "2021-02-15T15:00:00Z", dChild.Attribute("stamp"))
	require.Equal(t, "Delayed IQ", dChild.Text())
}
