/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

const clusterOpTimeout = time.Millisecond * 250

type fakeClusterDelegate struct {
	nodeJoinedCalls    int
	nodeUpdatedCalls   int
	nodeLeftCalls      int
	notifyMessageCalls int
}

func (d *fakeClusterDelegate) NodeJoined(node *Node)      { d.nodeJoinedCalls++ }
func (d *fakeClusterDelegate) NodeUpdated(node *Node)     { d.nodeUpdatedCalls++ }
func (d *fakeClusterDelegate) NodeLeft(node *Node)        { d.nodeLeftCalls++ }
func (d *fakeClusterDelegate) NotifyMessage(msg *Message) { d.notifyMessageCalls++ }

type fakeMemberList struct {
	members           []Node
	joinHosts         []string
	sendErr           error
	sendCh            chan []byte
	shutdownCh        chan struct{}
	membersCalls      int
	joinCalls         int
	shutdownCalls     int
	sendReliableCalls int
}

func (ml *fakeMemberList) Members() []Node {
	ml.membersCalls++
	return ml.members
}

func (ml *fakeMemberList) Join(hosts []string) error {
	ml.joinHosts = hosts
	ml.joinCalls++
	return nil
}

func (ml *fakeMemberList) Shutdown() error {
	if ml.shutdownCh != nil {
		close(ml.shutdownCh)
	}
	ml.shutdownCalls++
	return nil
}

func (ml *fakeMemberList) SendReliable(node string, msg []byte) error {
	if ml.sendErr != nil {
		return ml.sendErr
	}
	if ml.sendCh != nil {
		ml.sendCh <- msg
	}
	ml.sendReliableCalls++
	return nil
}

func TestCluster_Create(t *testing.T) {
	var ml fakeMemberList
	createMemberList = func(_ string, _ int, _ *Cluster) (list memberList, e error) {
		return &ml, nil
	}
	c, _ := New(nil, nil)
	require.Nil(t, c)

	c, _ = New(testClusterConfig(), nil)
	require.NotNil(t, c)
	require.Equal(t, "node1", c.LocalNode())
}

func TestCluster_Shutdown(t *testing.T) {
	var ml fakeMemberList
	createMemberList = func(_ string, _ int, _ *Cluster) (list memberList, e error) {
		return &ml, nil
	}
	c, _ := New(testClusterConfig(), nil)
	require.NotNil(t, c)

	ml.shutdownCh = make(chan struct{})

	_ = c.Shutdown()
	select {
	case <-ml.shutdownCh:
		break
	case <-time.After(clusterOpTimeout):
		require.Fail(t, "cluster shutdown timeout")
	}
}

func TestCluster_Join(t *testing.T) {
	var ml fakeMemberList
	createMemberList = func(_ string, _ int, _ *Cluster) (list memberList, e error) {
		return &ml, nil
	}
	c, _ := New(testClusterConfig(), nil)
	require.NotNil(t, c)

	ml.members = []Node{{Name: "node2"}, {Name: "node3"}}
	err := c.Join()
	require.Nil(t, err)

	require.Equal(t, 1, ml.membersCalls)
	require.Equal(t, 1, ml.joinCalls)

	require.Equal(t, 2, len(ml.joinHosts))
}

func TestCluster_SendAndBroadcast(t *testing.T) {
	var ml fakeMemberList
	createMemberList = func(_ string, _ int, _ *Cluster) (list memberList, e error) {
		return &ml, nil
	}
	c, _ := New(testClusterConfig(), nil)
	require.NotNil(t, c)

	ml.members = []Node{{Name: "node2"}, {Name: "node3"}}
	err := c.Join()
	require.Nil(t, err)

	ml.sendCh = make(chan []byte)
	c.SendMessageTo("node3", &Message{})
	select {
	case <-ml.sendCh:
		break
	case <-time.After(clusterOpTimeout):
		require.Fail(t, "cluster send message timeout")
	}

	c.BroadcastMessage(&Message{})

	for i := 0; i < 2; i++ {
		select {
		case <-ml.sendCh:
			break
		case <-time.After(clusterOpTimeout):
			require.Fail(t, "cluster broadcast message timeout")
		}
	}

	// test send error
	ml.sendErr = errors.New("cluster: send error")

	c.SendMessageTo("node3", &Message{})
	select {
	case <-ml.sendCh:
		require.Fail(t, "unexpected send message")
	case <-time.After(clusterOpTimeout):
		break
	}

	c.BroadcastMessage(&Message{})

	for i := 0; i < 2; i++ {
		select {
		case <-ml.sendCh:
			require.Fail(t, "unexpected broadcast message")
		case <-time.After(clusterOpTimeout):
			break
		}
	}
}

func TestCluster_Delegate(t *testing.T) {
	var ml fakeMemberList
	var delegate fakeClusterDelegate

	createMemberList = func(_ string, _ int, _ *Cluster) (list memberList, e error) {
		return &ml, nil
	}
	c, _ := New(testClusterConfig(), &delegate)
	require.NotNil(t, c)

	c.handleNotifyJoin(&Node{Name: "node4"})
	require.Equal(t, 1, delegate.nodeJoinedCalls)

	c.handleNotifyUpdate(&Node{Name: "node4"})
	require.Equal(t, 1, delegate.nodeUpdatedCalls)

	c.handleNotifyLeave(&Node{Name: "node4"})
	require.Equal(t, 1, delegate.nodeLeftCalls)

	j, _ := jid.NewWithString("ortuman@jackal.im/garden", true)
	var m Message
	m.Type = MsgBind
	m.Node = "node3"
	m.Payloads = []MessagePayload{{JID: j}}

	buf := bytes.NewBuffer(nil)
	require.Nil(t, m.ToBytes(buf))

	c.handleNotifyMsg(buf.Bytes())
	require.Equal(t, 1, delegate.notifyMessageCalls)
}

func testClusterConfig() *Config {
	return &Config{
		Name:     "node1",
		BindPort: 9999,
		Hosts:    []string{"127.0.0.1:6666", "127.0.0.1:7777"},
	}
}
