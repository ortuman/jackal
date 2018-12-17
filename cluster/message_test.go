/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestMessageSerialization(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	var m1, m2 Message
	m1 = Message{
		Type: MsgBatchBind,
		Node: "node1",
	}
	m1.ToGob(gob.NewEncoder(buf))

	require.Nil(t, m2.FromGob(gob.NewDecoder(buf)))
	require.Equal(t, m1.Type, m2.Type)
	require.Equal(t, m1.Node, m2.Node)

	j, _ := jid.NewWithString("ortuman@jackal.im", true)
	m1 = Message{
		Type: MsgUpdatePresence,
		Node: "node1",
		Payloads: []MessagePayload{{
			JID:     j,
			Stanza:  xmpp.NewPresence(j, j, xmpp.UnavailableType),
			Context: map[string]interface{}{"requested": true},
		}},
	}
	buf.Reset()
	m1.ToGob(gob.NewEncoder(buf))

	require.Nil(t, m2.FromGob(gob.NewDecoder(buf)))
	require.Equal(t, m1.Type, m2.Type)
	require.Equal(t, m1.Node, m2.Node)
	require.Equal(t, 1, len(m2.Payloads))
	require.NotNil(t, m2.Payloads[0].JID)
	require.NotNil(t, m2.Payloads[0].Stanza)
	require.NotNil(t, m2.Payloads[0].Context)

	require.Equal(t, m1.Payloads[0].Context, m2.Payloads[0].Context)
	require.Equal(t, m1.Payloads[0].JID.String(), m2.Payloads[0].JID.String())
	require.Equal(t, m1.Payloads[0].Stanza.String(), m2.Payloads[0].Stanza.String())
	_, ok := m2.Payloads[0].Stanza.(*xmpp.Presence)
	require.True(t, ok)

	m1.Payloads[0].Stanza = xmpp.NewIQType(uuid.New().String(), xmpp.GetType)
	buf.Reset()
	m1.ToGob(gob.NewEncoder(buf))

	require.Nil(t, m2.FromGob(gob.NewDecoder(buf)))
	_, ok = m2.Payloads[0].Stanza.(*xmpp.IQ)
	require.True(t, ok)

	m1.Payloads[0].Stanza = xmpp.NewMessageType(uuid.New().String(), xmpp.NormalType)
	buf.Reset()
	m1.ToGob(gob.NewEncoder(buf))

	require.Nil(t, m2.FromGob(gob.NewDecoder(buf)))
	_, ok = m2.Payloads[0].Stanza.(*xmpp.Message)
	require.True(t, ok)
}
