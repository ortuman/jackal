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

type defaultMemberList struct {
	cluster *Cluster
	ml      *memberlist.Memberlist
	mu      sync.RWMutex
	members map[string]*memberlist.Node
}

func newDefaultMemberList(config *Config, c *Cluster) (MemberList, error) {
	dl := &defaultMemberList{
		cluster: c,
		members: make(map[string]*memberlist.Node),
	}
	conf := memberlist.DefaultLocalConfig()
	conf.Name = config.Name
	conf.BindPort = config.BindPort
	conf.Delegate = dl
	conf.Events = dl
	conf.LogOutput = ioutil.Discard
	ml, err := memberlist.Create(conf)
	if err != nil {
		return nil, err
	}
	dl.ml = ml
	return dl, nil
}

func (d *defaultMemberList) Members() []Node {
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

func (d *defaultMemberList) Join(hosts []string) error {
	_, err := d.ml.Join(hosts)
	return err
}

func (d *defaultMemberList) Shutdown() error {
	if err := d.ml.Leave(leaveTimeout); err != nil {
		return err
	}
	return d.ml.Shutdown()
}

func (d *defaultMemberList) SendReliable(toNode string, msg []byte) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	node := d.members[toNode]
	if node == nil {
		return fmt.Errorf("cannot send message: node %s not found", toNode)
	}
	return d.ml.SendReliable(node, msg)
}

// memberlist.Delegate

func (d *defaultMemberList) NodeMeta(limit int) []byte {
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

func (d *defaultMemberList) NotifyMsg(msg []byte)                       { d.cluster.handleNotifyMsg(msg) }
func (d *defaultMemberList) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (d *defaultMemberList) LocalState(join bool) []byte                { return nil }
func (d *defaultMemberList) MergeRemoteState(buf []byte, join bool)     {}

// memberlist.EventDelegate

func (d *defaultMemberList) NotifyJoin(n *memberlist.Node) {
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

func (d *defaultMemberList) NotifyLeave(n *memberlist.Node) {
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

func (d *defaultMemberList) NotifyUpdate(n *memberlist.Node) {}

func (d *defaultMemberList) clusterNodeFromMemberListNode(n *memberlist.Node) (*Node, error) {
	var m Metadata
	if err := gob.NewDecoder(bytes.NewBuffer(n.Meta)).Decode(&m); err != nil {
		return nil, err
	}
	return &Node{
		Name:     n.Name,
		Metadata: m,
	}, nil
}
