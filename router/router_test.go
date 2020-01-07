/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"context"
	"crypto/tls"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ortuman/jackal/cluster"
	"github.com/ortuman/jackal/model"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

const routerOpTimeout = time.Millisecond * 250

type fakeClusterDelegate struct {
	cluster               cluster.Cluster
	sendCh                chan *cluster.Message
	sendMessageToCalls    int
	broadcastMessageCalls int
}

func (d *fakeClusterDelegate) LocalNode() string {
	return "node1"
}

func (d *fakeClusterDelegate) C2SStream(jid *jid.JID, presence *xmpp.Presence, context map[string]interface{}, node string) *cluster.C2S {
	return d.cluster.C2SStream(jid, presence, context, node)
}

func (d *fakeClusterDelegate) SendMessageTo(_ context.Context, _ string, message *cluster.Message) {
	if d.sendCh != nil {
		d.sendCh <- message
	}
	d.sendMessageToCalls++
}

func (d *fakeClusterDelegate) BroadcastMessage(_ context.Context, _ *cluster.Message) {
	d.broadcastMessageCalls++
}

type fakeS2SOut struct {
	elems []xmpp.XElement
}

func (f *fakeS2SOut) ID() string { return uuid.New() }
func (f *fakeS2SOut) SendElement(_ context.Context, elem xmpp.XElement) {
	f.elems = append(f.elems, elem)
}
func (f *fakeS2SOut) Disconnect(_ context.Context, _ error) {}

type fakeOutS2SProvider struct{ s2sOut *fakeS2SOut }

func (f *fakeOutS2SProvider) GetOut(_ context.Context, _, _ string) (stream.S2SOut, error) {
	return f.s2sOut, nil
}

func TestRouter_EmptyConfig(t *testing.T) {
	defer func() { _ = os.RemoveAll("./.cert") }()

	r, _ := New(&Config{}, nil, nil)
	require.True(t, r.IsLocalHost("localhost"))
	require.Equal(t, 1, len(r.HostNames()))
	require.Equal(t, 1, len(r.Certificates()))
}

func TestRouter_SetCluster(t *testing.T) {
	r, _, _ := setupTest()

	var del fakeClusterDelegate
	r.SetCluster(&del)
	require.Equal(t, &del, r.Cluster())
}

func TestRouter_ClusterDelegate(t *testing.T) {
	r, _, _ := setupTest()

	del, ok := r.ClusterDelegate().(cluster.Delegate)
	require.True(t, ok)
	require.NotNil(t, del)
}

func TestRouter_Binding(t *testing.T) {
	r, _, _ := setupTest()

	var del fakeClusterDelegate
	r.SetCluster(&del)

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("ortuman@jackal.im/garden", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j4, _ := jid.NewWithString("romeo@jackal.im/balcony", false)
	j5, _ := jid.NewWithString("juliet@jackal.im/garden", false)
	j6, _ := jid.NewWithString("juliet@jackal.im", false) // empty resource
	j7, _ := jid.NewWithString("juliet@jackal.im/yard", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm3 := stream.NewMockC2S(uuid.New(), j3)
	stm4 := stream.NewMockC2S(uuid.New(), j4)
	stm5 := stream.NewMockC2S(uuid.New(), j5)
	stm6 := stream.NewMockC2S(uuid.New(), j6)

	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)
	r.Bind(context.Background(), stm3)
	r.Bind(context.Background(), stm4)
	r.Bind(context.Background(), stm5)
	r.Bind(context.Background(), stm6)

	require.Equal(t, 5, del.broadcastMessageCalls)

	require.Equal(t, 2, len(r.UserStreams("ortuman")))
	require.Equal(t, 1, len(r.UserStreams("hamlet")))
	require.Equal(t, 1, len(r.UserStreams("romeo")))
	require.Equal(t, 1, len(r.UserStreams("juliet")))

	r.Unbind(context.Background(), j7)
	r.Unbind(context.Background(), j6)
	r.Unbind(context.Background(), j5)
	r.Unbind(context.Background(), j4)
	r.Unbind(context.Background(), j3)
	r.Unbind(context.Background(), j2)
	r.Unbind(context.Background(), j1)

	require.Equal(t, 10, del.broadcastMessageCalls)

	require.Equal(t, 0, len(r.UserStreams("ortuman")))
	require.Equal(t, 0, len(r.UserStreams("hamlet")))
	require.Equal(t, 0, len(r.UserStreams("romeo")))
	require.Equal(t, 0, len(r.UserStreams("juliet")))
}

func TestRouter_Routing(t *testing.T) {
	outS2S := fakeS2SOut{}
	s2sOutProvider := fakeOutS2SProvider{s2sOut: &outS2S}

	r, userRep, _ := setupTest()

	r.SetOutS2SProvider(&s2sOutProvider)

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("ortuman@jackal.im/garden", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j4, _ := jid.NewWithString("hamlet@jackal.im/garden", false)
	j5, _ := jid.NewWithString("hamlet@jackal.im", false)
	j6, _ := jid.NewWithString("juliet@example.org/garden", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm3 := stream.NewMockC2S(uuid.New(), j3)

	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j6)

	// remote routing
	require.Nil(t, r.Route(context.Background(), iq))
	require.Equal(t, 1, len(outS2S.elems))

	iq.SetToJID(j3)
	require.Equal(t, ErrNotExistingAccount, r.Route(context.Background(), iq))

	memorystorage.EnableMockedError()
	require.Equal(t, memorystorage.ErrMocked, r.Route(context.Background(), iq))
	memorystorage.DisableMockedError()

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "hamlet", Password: ""})
	require.Equal(t, ErrNotAuthenticated, r.Route(context.Background(), iq))

	stm4 := stream.NewMockC2S(uuid.New(), j4)
	r.Bind(context.Background(), stm4)
	require.Equal(t, ErrResourceNotFound, r.Route(context.Background(), iq))

	r.Bind(context.Background(), stm3)
	require.Nil(t, r.Route(context.Background(), iq))
	elem := stm3.ReceiveElement()
	require.Equal(t, iqID, elem.ID())

	// broadcast stanza
	iq.SetToJID(j5)
	require.Nil(t, r.Route(context.Background(), iq))
	elem = stm3.ReceiveElement()
	require.Equal(t, iqID, elem.ID())
	elem = stm4.ReceiveElement()
	require.Equal(t, iqID, elem.ID())

	// send clusterMessage to highest priority
	p1 := xmpp.NewElementName("presence")
	p1.SetFrom(j3.String())
	p1.SetTo(j3.String())
	p1.SetType(xmpp.AvailableType)
	pr1 := xmpp.NewElementName("priority")
	pr1.SetText("2")
	p1.AppendElement(pr1)
	presence1, _ := xmpp.NewPresenceFromElement(p1, j3, j3)
	stm3.SetPresence(presence1)

	p2 := xmpp.NewElementName("presence")
	p2.SetFrom(j4.String())
	p2.SetTo(j4.String())
	p2.SetType(xmpp.AvailableType)
	pr2 := xmpp.NewElementName("priority")
	pr2.SetText("1")
	p2.AppendElement(pr2)
	presence2, _ := xmpp.NewPresenceFromElement(p2, j4, j4)
	stm4.SetPresence(presence2)

	msgID := uuid.New()
	msg := xmpp.NewMessageType(msgID, xmpp.ChatType)
	msg.SetToJID(j5)
	require.Nil(t, r.Route(context.Background(), msg))
	elem = stm3.ReceiveElement()
	require.Equal(t, msgID, elem.ID())
}

func TestRouter_BlockedJID(t *testing.T) {
	r, _, blockListRep := setupTest()

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/garden", false)
	j4, _ := jid.NewWithString("juliet@jackal.im/garden", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)

	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)

	// node + domain + resource
	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	})
	require.False(t, r.IsBlockedJID(context.Background(), j2, "ortuman"))
	require.True(t, r.IsBlockedJID(context.Background(), j3, "ortuman"))

	_ = blockListRep.DeleteBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	})

	// node + domain
	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "hamlet@jackal.im",
	})
	r.ReloadBlockList("ortuman")

	require.True(t, r.IsBlockedJID(context.Background(), j2, "ortuman"))
	require.True(t, r.IsBlockedJID(context.Background(), j3, "ortuman"))
	require.False(t, r.IsBlockedJID(context.Background(), j4, "ortuman"))

	_ = blockListRep.DeleteBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "hamlet@jackal.im",
	})

	// domain + resource
	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "jackal.im/balcony",
	})
	r.ReloadBlockList("ortuman")

	require.True(t, r.IsBlockedJID(context.Background(), j2, "ortuman"))
	require.False(t, r.IsBlockedJID(context.Background(), j3, "ortuman"))
	require.False(t, r.IsBlockedJID(context.Background(), j4, "ortuman"))

	_ = blockListRep.DeleteBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "jackal.im/balcony",
	})

	// domain
	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "jackal.im",
	})
	r.ReloadBlockList("ortuman")

	require.True(t, r.IsBlockedJID(context.Background(), j2, "ortuman"))
	require.True(t, r.IsBlockedJID(context.Background(), j3, "ortuman"))
	require.True(t, r.IsBlockedJID(context.Background(), j4, "ortuman"))

	_ = blockListRep.DeleteBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "jackal.im",
	})

	// test blocked routing
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1)
	require.Equal(t, ErrBlockedJID, r.Route(context.Background(), iq))
}

func TestRouter_Cluster(t *testing.T) {
	r, _, _ := setupTest()

	var del fakeClusterDelegate
	del.sendCh = make(chan *cluster.Message, 2)
	r.SetCluster(&del)

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("ortuman@jackal.im/garden", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm3 := stream.NewMockC2S(uuid.New(), j3)

	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)
	r.Bind(context.Background(), stm3)

	node := &cluster.Node{
		Name: "node2",
		Metadata: cluster.Metadata{
			Version:   version.ApplicationVersion.String(),
			GoVersion: runtime.Version(),
		},
	}
	bindMsgBatchSize = 2

	r.handleNodeJoined(context.Background(), node)

	// expecting 2 batches
	for i := 0; i < 2; i++ {
		select {
		case <-del.sendCh:
			break
		case <-time.After(routerOpTimeout):
			require.Fail(t, "handle cluster join timeout")
		}
	}
	require.Equal(t, 2, del.sendMessageToCalls)

	// try to join with incompatible version
	r.handleNodeJoined(context.Background(), &cluster.Node{
		Name: "node3",
		Metadata: cluster.Metadata{
			Version:   version.ApplicationVersion.String(),
			GoVersion: "v0.1",
		},
	})
	r.handleNodeJoined(context.Background(), &cluster.Node{
		Name: "node4",
		Metadata: cluster.Metadata{
			Version:   "v0.0.0.1.rc2",
			GoVersion: runtime.Version(),
		},
	})
	require.Equal(t, 2, del.sendMessageToCalls) // nothing happened

	r.SetCluster(nil)
	r.handleNodeJoined(context.Background(), node)
	require.Equal(t, 2, del.sendMessageToCalls) // nothing happened

	// process bind message
	r.SetCluster(&del)

	j4, _ := jid.NewWithString("noelia@jackal.im/balcony", true)
	j5, _ := jid.NewWithString("noelia@jackal.im/yard", true)

	r.handleNotifyMessage(context.Background(), &cluster.Message{
		Type: cluster.MsgBind,
		Node: "node2",
		Payloads: []cluster.MessagePayload{{
			JID:     j4,
			Stanza:  xmpp.NewPresence(j4, j4, xmpp.AvailableType),
			Context: map[string]interface{}{},
		}},
	})
	r.handleNotifyMessage(context.Background(), &cluster.Message{
		Type: cluster.MsgBind,
		Node: "node2",
		Payloads: []cluster.MessagePayload{{
			JID:     j5,
			Stanza:  xmpp.NewPresence(j5, j5, xmpp.AvailableType),
			Context: map[string]interface{}{},
		}},
	})
	r.mu.RLock()
	require.Equal(t, 2, len(r.clusterStreams["node2"]))
	r.mu.RUnlock()

	r.handleNotifyMessage(context.Background(), &cluster.Message{
		Type: cluster.MsgUnbind,
		Node: "node2",
		Payloads: []cluster.MessagePayload{{
			JID:    j5,
			Stanza: xmpp.NewPresence(j5, j5, xmpp.AvailableType),
		}},
	})
	r.mu.RLock()
	require.Equal(t, 1, len(r.clusterStreams["node2"]))
	r.mu.RUnlock()

	// update cluster stream presence
	p := xmpp.NewPresence(j4, j4, xmpp.UnavailableType)
	r.handleNotifyMessage(context.Background(), &cluster.Message{
		Type: cluster.MsgUpdatePresence,
		Node: "node2",
		Payloads: []cluster.MessagePayload{{
			JID:    j4,
			Stanza: p,
		}},
	})
	r.mu.RLock()
	stm := r.clusterStreams["node2"][j4.String()]
	require.NotNil(t, stm)
	require.Equal(t, stm.Presence(), p)
	r.mu.RUnlock()

	// update cluster stream context
	r.handleNotifyMessage(context.Background(), &cluster.Message{
		Type: cluster.MsgUpdateContext,
		Node: "node2",
		Payloads: []cluster.MessagePayload{{
			JID: j4,
			Context: map[string]interface{}{
				"var": "foo",
			},
		}},
	})
	r.mu.RLock()
	stm = r.clusterStreams["node2"][j4.String()]
	require.NotNil(t, stm)
	require.Equal(t, "foo", stm.GetString("var"))
	r.mu.RUnlock()

	r.handleNodeLeft(context.Background(), &cluster.Node{
		Name: "node2",
		Metadata: cluster.Metadata{
			Version:   version.ApplicationVersion.String(),
			GoVersion: runtime.Version(),
		},
	})
	r.mu.RLock()
	require.Equal(t, 0, len(r.clusterStreams["node2"]))
	r.mu.RUnlock()

	// test cluster stanza routing
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j4)
	iq.SetToJID(j3)

	r.handleNotifyMessage(context.Background(), &cluster.Message{
		Type: cluster.MsgRouteStanza,
		Node: "node2",
		Payloads: []cluster.MessagePayload{{
			JID:    j4,
			Stanza: iq,
		}},
	})
	elem := stm3.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, elem, iq)
}

func setupTest() (*Router, repository.User, repository.BlockList) {
	userRep := memorystorage.NewUser()
	blockListRep := memorystorage.NewBlockList()
	r, _ := New(
		&Config{
			Hosts: []HostConfig{{Name: "jackal.im", Certificate: tls.Certificate{}}},
		},
		userRep,
		blockListRep,
	)
	return r, userRep, blockListRep
}
