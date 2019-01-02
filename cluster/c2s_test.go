/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

type fakeC2SCluster struct {
	sendMessageToCalls int
}

func (c *fakeC2SCluster) LocalNode() string                       { return "node1" }
func (c *fakeC2SCluster) SendMessageTo(node string, msg *Message) { c.sendMessageToCalls++ }

func TestC2S_New(t *testing.T) {
	var c fakeC2SCluster

	id := uuid.New().String()
	stm := newTestClusterC2S(id, "ortuman@jackal.im/balcony", xmpp.AvailableType, map[string]interface{}{}, "node1", &c)

	require.Equal(t, id, stm.ID())
	require.True(t, stm.IsSecured())
	require.True(t, stm.IsAuthenticated())

	require.Equal(t, "ortuman", stm.Username())
	require.Equal(t, "jackal.im", stm.Domain())
	require.Equal(t, "balcony", stm.Resource())

	j := stm.JID()
	require.NotNil(t, j)
	require.Equal(t, "ortuman", j.Node())
	require.Equal(t, "jackal.im", j.Domain())
	require.Equal(t, "balcony", j.Resource())
}

func TestC2S_Presence(t *testing.T) {
	var c fakeC2SCluster

	id := uuid.New().String()
	stm := newTestClusterC2S(id, "ortuman@jackal.im/balcony", xmpp.AvailableType, map[string]interface{}{}, "node1", &c)

	p := stm.Presence()
	require.NotNil(t, p)
	require.Equal(t, xmpp.AvailableType, p.Type())

	// change presence
	p = xmpp.NewPresence(p.FromJID(), p.ToJID(), xmpp.UnavailableType)
	stm.SetPresence(p)
	require.Equal(t, p, stm.Presence())
}

func TestC2S_Context(t *testing.T) {
	var c fakeC2SCluster

	context := map[string]interface{}{
		"a1": true,
		"b1": 3.14,
		"c1": 35,
		"d1": "foo",
	}
	contextLength := len(context)

	id := uuid.New().String()
	stm := newTestClusterC2S(id, "ortuman@jackal.im/balcony", xmpp.AvailableType, context, "node1", &c)

	// setters don't do anything
	stm.SetBool("a2", true)
	stm.SetFloat("b2", 3.14)
	stm.SetInt("c2", 35)
	stm.SetString("d2", "foo")

	require.Equal(t, contextLength, len(stm.Context()))

	require.True(t, stm.GetBool("a1"))
	require.Equal(t, 3.14, stm.GetFloat("b1"))
	require.Equal(t, 35, stm.GetInt("c1"))
	require.Equal(t, "foo", stm.GetString("d1"))

	// update context
	stm.UpdateContext(map[string]interface{}{
		"e1": "foo2",
	})

	require.Equal(t, contextLength+1, len(stm.Context()))
	require.Equal(t, "foo2", stm.GetString("e1"))
}

func TestC2S_SendElement(t *testing.T) {
	var c fakeC2SCluster

	id := uuid.New().String()
	stm := newTestClusterC2S(id, "ortuman@jackal.im/balcony", xmpp.AvailableType, map[string]interface{}{}, "node1", &c)

	stm.SendElement(xmpp.NewElementName("vCard")) // not a stanza
	stm.SendElement(xmpp.NewIQType(uuid.New().String(), xmpp.GetType))

	require.Equal(t, 1, c.sendMessageToCalls)
}

func newTestClusterC2S(id string, jidString string, presenceType string, context map[string]interface{}, node string, c2sCluster c2sCluster) *C2S {
	j, _ := jid.NewWithString(jidString, true)
	p := xmpp.NewPresence(j, j, xmpp.AvailableType)
	return newC2S(id, j, p, context, node, c2sCluster)
}
