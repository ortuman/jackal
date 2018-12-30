/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"encoding/gob"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/version"
	"github.com/stretchr/testify/require"
)

type fakeHashicorpMemberList struct {
	err               error
	joinCalls         int32
	leaveCalls        int32
	shutdownCalls     int32
	sendReliableCalls int32
}

func (ml *fakeHashicorpMemberList) Join(existing []string) (int, error) {
	if ml.err != nil {
		return 0, ml.err
	}
	atomic.AddInt32(&ml.joinCalls, 1)
	return len(existing), nil
}

func (ml *fakeHashicorpMemberList) Leave(timeout time.Duration) error {
	if ml.err != nil {
		return ml.err
	}
	atomic.AddInt32(&ml.leaveCalls, 1)
	return nil
}

func (ml *fakeHashicorpMemberList) Shutdown() error {
	if ml.err != nil {
		return ml.err
	}
	atomic.AddInt32(&ml.shutdownCalls, 1)
	return nil
}

func (ml *fakeHashicorpMemberList) SendReliable(to *memberlist.Node, msg []byte) error {
	if ml.err != nil {
		return ml.err
	}
	atomic.AddInt32(&ml.sendReliableCalls, 1)
	return nil
}

type fakeMemberListDelegate struct {
	notifyMsgCalls    int32
	notifyJoinCalls   int32
	notifyUpdateCalls int32
	notifyLeaveCalls  int32
}

func (d *fakeMemberListDelegate) handleNotifyMsg(msg []byte) {
	atomic.AddInt32(&d.notifyMsgCalls, 1)
}

func (d *fakeMemberListDelegate) handleNotifyJoin(n *Node) {
	atomic.AddInt32(&d.notifyJoinCalls, 1)
}

func (d *fakeMemberListDelegate) handleNotifyUpdate(n *Node) {
	atomic.AddInt32(&d.notifyUpdateCalls, 1)
}

func (d *fakeMemberListDelegate) handleNotifyLeave(n *Node) {
	atomic.AddInt32(&d.notifyLeaveCalls, 1)
}

func TestClusterMemberList_Members(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, &delegate)
	cMemberList.NotifyJoin(memberListNode("node1"))
	cMemberList.NotifyJoin(memberListNode("node2"))
	cMemberList.NotifyJoin(memberListNode("node3"))

	// no metadata included... node won't be added
	cMemberList.NotifyJoin(&memberlist.Node{Name: "node4"})

	require.Equal(t, int32(3), atomic.LoadInt32(&delegate.notifyJoinCalls))

	cMemberList.NotifyUpdate(&memberlist.Node{Name: "node2"})
	cMemberList.NotifyUpdate(memberListNode("node2"))

	require.Equal(t, int32(1), atomic.LoadInt32(&delegate.notifyUpdateCalls))

	cMemberList.NotifyLeave(&memberlist.Node{Name: "node3"})
	cMemberList.NotifyLeave(memberListNode("node3"))

	require.Equal(t, int32(1), atomic.LoadInt32(&delegate.notifyLeaveCalls))

	require.Equal(t, 2, len(cMemberList.Members()))
}

func TestClusterMemberList_Join(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, &delegate)

	err := cMemberList.Join([]string{"127.0.0.1:7777", "127.0.0.1:8888"})
	require.Nil(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.joinCalls))

	ml.err = errors.New("")
	err = cMemberList.Join([]string{"127.0.0.1:7777", "127.0.0.1:8888"})
	require.NotNil(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.joinCalls))
}

func TestClusterMemberList_Shutdown(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, &delegate)
	err := cMemberList.Shutdown()
	require.Nil(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.leaveCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.shutdownCalls))

	ml.err = errors.New("")
	err = cMemberList.Shutdown()
	require.NotNil(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.leaveCalls))
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.shutdownCalls))
}

func TestClusterMemberList_SendReliable(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, &delegate)
	err := cMemberList.SendReliable("node2", []byte{})
	require.NotNil(t, err) // node2 has not joined
	require.Equal(t, int32(0), atomic.LoadInt32(&ml.sendReliableCalls))

	cMemberList.NotifyJoin(memberListNode("node2")) // node2 joins

	err = cMemberList.SendReliable("node2", []byte{})
	require.Nil(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.sendReliableCalls))

	ml.err = errors.New("")
	err = cMemberList.SendReliable("node2", []byte{})
	require.NotNil(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&ml.sendReliableCalls))
}

func TestClusterMemberList_NodeMetadata(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, &delegate)
	require.Nil(t, cMemberList.NodeMeta(1))

	b := cMemberList.NodeMeta(10000)
	var meta Metadata
	_ = gob.NewDecoder(bytes.NewReader(b)).Decode(&meta)

	require.Equal(t, meta.Version, version.ApplicationVersion.String())
	require.Equal(t, meta.GoVersion, runtime.Version())
}

func memberListNode(name string) *memberlist.Node {
	var m Metadata
	m.Version = version.ApplicationVersion.String()
	m.GoVersion = runtime.Version()

	buf := bytes.NewBuffer(nil)
	_ = gob.NewEncoder(buf).Encode(&m)
	b := make([]byte, buf.Len())
	copy(b, buf.Bytes())
	return &memberlist.Node{
		Name: name,
		Meta: b,
	}
}
