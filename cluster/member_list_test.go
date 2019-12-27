/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/version"
	"github.com/stretchr/testify/require"
)

type fakeHashicorpMemberList struct {
	err               error
	joinCalls         int
	leaveCalls        int
	shutdownCalls     int
	sendReliableCalls int
}

func (ml *fakeHashicorpMemberList) Join(existing []string) (int, error) {
	if ml.err != nil {
		return 0, ml.err
	}
	ml.joinCalls++
	return len(existing), nil
}

func (ml *fakeHashicorpMemberList) Leave(timeout time.Duration) error {
	if ml.err != nil {
		return ml.err
	}
	ml.leaveCalls++
	return nil
}

func (ml *fakeHashicorpMemberList) Shutdown() error {
	if ml.err != nil {
		return ml.err
	}
	ml.shutdownCalls++
	return nil
}

func (ml *fakeHashicorpMemberList) SendReliable(to *memberlist.Node, msg []byte) error {
	if ml.err != nil {
		return ml.err
	}
	ml.sendReliableCalls++
	return nil
}

type fakeMemberListDelegate struct {
	notifyMsgCalls    int
	notifyJoinCalls   int
	notifyUpdateCalls int
	notifyLeaveCalls  int
}

func (d *fakeMemberListDelegate) handleNotifyMsg(_ context.Context, _ []byte)   { d.notifyMsgCalls++ }
func (d *fakeMemberListDelegate) handleNotifyJoin(_ context.Context, _ *Node)   { d.notifyJoinCalls++ }
func (d *fakeMemberListDelegate) handleNotifyUpdate(_ context.Context, _ *Node) { d.notifyUpdateCalls++ }
func (d *fakeMemberListDelegate) handleNotifyLeave(_ context.Context, _ *Node)  { d.notifyLeaveCalls++ }

func TestClusterMemberList_Members(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, time.Minute, &delegate)
	cMemberList.NotifyJoin(memberListNode("node1"))
	cMemberList.NotifyJoin(memberListNode("node2"))
	cMemberList.NotifyJoin(memberListNode("node3"))

	// no metadata included... node won't be added
	cMemberList.NotifyJoin(&memberlist.Node{Name: "node4"})

	require.Equal(t, 3, delegate.notifyJoinCalls)

	cMemberList.NotifyUpdate(&memberlist.Node{Name: "node2"})
	cMemberList.NotifyUpdate(memberListNode("node2"))

	require.Equal(t, 1, delegate.notifyUpdateCalls)

	cMemberList.NotifyLeave(&memberlist.Node{Name: "node3"})
	cMemberList.NotifyLeave(memberListNode("node3"))

	require.Equal(t, 1, delegate.notifyLeaveCalls)

	require.Equal(t, 2, len(cMemberList.Members()))
}

func TestClusterMemberList_Join(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, time.Minute, &delegate)

	err := cMemberList.Join([]string{"127.0.0.1:7777", "127.0.0.1:8888"})
	require.Nil(t, err)
	require.Equal(t, 1, ml.joinCalls)

	ml.err = errors.New("")
	err = cMemberList.Join([]string{"127.0.0.1:7777", "127.0.0.1:8888"})
	require.NotNil(t, err)
	require.Equal(t, 1, ml.joinCalls)
}

func TestClusterMemberList_Shutdown(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, time.Minute, &delegate)
	err := cMemberList.Shutdown()
	require.Nil(t, err)
	require.Equal(t, 1, ml.leaveCalls)
	require.Equal(t, 1, ml.shutdownCalls)

	ml.err = errors.New("")
	err = cMemberList.Shutdown()
	require.NotNil(t, err)
	require.Equal(t, 1, ml.leaveCalls)
	require.Equal(t, 1, ml.shutdownCalls)
}

func TestClusterMemberList_SendReliable(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, time.Minute, &delegate)
	err := cMemberList.SendReliable("node2", []byte{})
	require.NotNil(t, err) // node2 has not joined
	require.Equal(t, 0, ml.sendReliableCalls)

	cMemberList.NotifyJoin(memberListNode("node2")) // node2 joins

	err = cMemberList.SendReliable("node2", []byte{})
	require.Nil(t, err)
	require.Equal(t, 1, ml.sendReliableCalls)

	ml.err = errors.New("")
	err = cMemberList.SendReliable("node2", []byte{})
	require.NotNil(t, err)
	require.Equal(t, 1, ml.sendReliableCalls)
}

func TestClusterMemberList_NodeMetadata(t *testing.T) {
	var ml fakeHashicorpMemberList
	var delegate fakeMemberListDelegate

	createHashicorpMemberList = func(_ *memberlist.Config) (list hashicorpMemberList, e error) {
		return &ml, nil
	}
	cMemberList, _ := newDefaultMemberList("node1", 6666, time.Minute, &delegate)
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
