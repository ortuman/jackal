/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"runtime"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/version"
)

const leaveTimeout = time.Second * 5

var createHashicorpMemberList = func(conf *memberlist.Config) (hashicorpMemberList, error) {
	return memberlist.Create(conf)
}

type hashicorpMemberList interface {
	Join(existing []string) (int, error)

	Leave(timeout time.Duration) error
	Shutdown() error

	SendReliable(to *memberlist.Node, msg []byte) error
}

type clusterMemberList struct {
	cluster *Cluster
	ml      hashicorpMemberList
	mu      sync.RWMutex
	members map[string]*memberlist.Node
}

func newDefaultMemberList(localName string, bindPort int, c *Cluster) (MemberList, error) {
	dl := &clusterMemberList{
		cluster: c,
		members: make(map[string]*memberlist.Node),
	}
	conf := memberlist.DefaultLocalConfig()
	conf.Name = localName
	conf.BindPort = bindPort
	conf.Delegate = dl
	conf.Events = dl
	conf.LogOutput = ioutil.Discard

	ml, err := createHashicorpMemberList(conf)
	if err != nil {
		return nil, err
	}
	dl.ml = ml
	return dl, nil
}

func (d *clusterMemberList) Members() []Node {
	var res []Node
	d.mu.RLock()
	for _, n := range d.members {
		cNode, err := d.clusterNodeFromMemberListNode(n)
		if err != nil {
			log.Warnf("%s", err)
			continue
		}
		res = append(res, *cNode)
	}
	d.mu.RUnlock()
	return res
}

func (d *clusterMemberList) Join(hosts []string) error {
	_, err := d.ml.Join(hosts)
	return err
}

func (d *clusterMemberList) Shutdown() error {
	if err := d.ml.Leave(leaveTimeout); err != nil {
		return err
	}
	return d.ml.Shutdown()
}

func (d *clusterMemberList) SendReliable(toNode string, msg []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	node := d.members[toNode]
	if node == nil {
		return fmt.Errorf("cannot send message: node %s not found", toNode)
	}
	return d.ml.SendReliable(node, msg)
}

// memberlist.Delegate

func (d *clusterMemberList) NodeMeta(limit int) []byte {
	var m Metadata
	m.Version = version.ApplicationVersion.String()
	m.GoVersion = runtime.Version()

	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(&m); err != nil {
		log.Error(err)
		return nil
	}
	b := make([]byte, buf.Len())
	copy(b, buf.Bytes())
	return b
}

func (d *clusterMemberList) NotifyMsg(msg []byte) {
	d.cluster.handleNotifyMsg(msg)
}

func (d *clusterMemberList) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (d *clusterMemberList) LocalState(join bool) []byte                { return nil }
func (d *clusterMemberList) MergeRemoteState(buf []byte, join bool)     {}

// memberlist.EventDelegate

func (d *clusterMemberList) NotifyJoin(n *memberlist.Node) {
	d.mu.Lock()
	d.members[n.Name] = n
	d.mu.Unlock()

	cNode, err := d.clusterNodeFromMemberListNode(n)
	if err != nil {
		log.Warnf("%s", err)
		return
	}
	d.cluster.handleNotifyJoin(cNode)
}

func (d *clusterMemberList) NotifyLeave(n *memberlist.Node) {
	d.mu.Lock()
	delete(d.members, n.Name)
	d.mu.Unlock()

	cNode, err := d.clusterNodeFromMemberListNode(n)
	if err != nil {
		log.Warnf("%s", err)
		return
	}
	d.cluster.handleNotifyLeave(cNode)
}

func (d *clusterMemberList) NotifyUpdate(n *memberlist.Node) {
	d.mu.Lock()
	d.members[n.Name] = n
	d.mu.Unlock()

	cNode, err := d.clusterNodeFromMemberListNode(n)
	if err != nil {
		log.Warnf("%s", err)
		return
	}
	d.cluster.handleNotifyUpdate(cNode)
}

func (d *clusterMemberList) clusterNodeFromMemberListNode(n *memberlist.Node) (*Node, error) {
	var m Metadata
	if err := gob.NewDecoder(bytes.NewBuffer(n.Meta)).Decode(&m); err != nil {
		return nil, err
	}
	return &Node{
		Name:     n.Name,
		Metadata: m,
	}, nil
}
