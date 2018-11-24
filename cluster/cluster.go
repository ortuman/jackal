/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
)

const leaveTimeout = time.Second * 5

type Node struct {
	Name string
}

type Delegate interface {
	NotifyMessage([]byte)
	NodeJoined(node *Node)
	NodeUpdated(node *Node)
	NodeLeft(node *Node)
}

type Cluster struct {
	cfg        *Config
	delegate   Delegate
	memberList *memberlist.Memberlist

	membersMu sync.RWMutex
	members   map[string]*memberlist.Node

	broadcastQueue *memberlist.TransmitLimitedQueue
}

func New(config *Config, delegate Delegate) (*Cluster, error) {
	if config == nil {
		return nil, nil
	}
	c := &Cluster{
		members: make(map[string]*memberlist.Node),
	}
	c.cfg = config
	c.delegate = delegate
	conf := memberlist.DefaultLocalConfig()
	conf.Name = config.Name
	conf.BindPort = config.BindPort
	conf.Delegate = &memberListDelegate{cluster: c}
	conf.Events = &memberListEventDelegate{cluster: c}
	conf.LogOutput = ioutil.Discard
	ml, err := memberlist.Create(conf)
	if err != nil {
		return nil, err
	}
	c.memberList = ml

	// setup broadcast queue
	c.broadcastQueue = &memberlist.TransmitLimitedQueue{
		NumNodes:       c.NumNodes,
		RetransmitMult: conf.RetransmitMult,
	}
	return c, nil
}

func (c *Cluster) Join() error {
	_, err := c.memberList.Join(c.cfg.Hosts)
	return err
}

func (c *Cluster) LocalNode() string {
	return c.memberList.LocalNode().Name
}

func (c *Cluster) Broadcast(msg []byte) error {
	c.broadcastQueue.QueueBroadcast(&broadcast{
		msg: msg,
	})
	return nil
}

func (c *Cluster) Send(msg []byte, toNode string) error {
	return nil
}

func (c *Cluster) Shutdown() error {
	if c.memberList != nil {
		if err := c.memberList.Leave(leaveTimeout); err != nil {
			return err
		}
		return c.memberList.Shutdown()
	}
	return nil
}

func (c *Cluster) NumNodes() int {
	c.membersMu.Lock()
	defer c.membersMu.Unlock()
	return len(c.members)
}

func (c *Cluster) handleNotifyJoin(n *memberlist.Node) {
	c.membersMu.Lock()
	c.members[n.Name] = n
	c.membersMu.Unlock()
	if c.delegate != nil {
		c.delegate.NodeJoined(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyLeave(n *memberlist.Node) {
	c.membersMu.Lock()
	delete(c.members, n.Name)
	c.membersMu.Unlock()
	if c.delegate != nil {
		c.delegate.NodeLeft(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyUpdate(n *memberlist.Node) {
	c.membersMu.Lock()
	c.members[n.Name] = n
	c.membersMu.Unlock()
	if c.delegate != nil {
		c.delegate.NodeUpdated(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyMsg(msg []byte) {
	if c.delegate != nil {
		c.delegate.NotifyMessage(msg)
	}
}
