/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"io/ioutil"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/xmpp"
)

const leaveTimeout = time.Second * 5

// messageType are the types of gossip messages jackal will send along
// memberlist.
type messageType uint8

const (
	messageBindType messageType = iota
	messageSendType
)

type Node struct {
	Name string
}

type Delegate interface {
	NodeJoined(node *Node)
	NodeLeft(node *Node)
}

type Cluster struct {
	cfg        *Config
	delegate   Delegate
	memberList *memberlist.Memberlist
}

func New(config *Config, delegate Delegate) (*Cluster, error) {
	if config == nil {
		return nil, nil
	}
	c := &Cluster{}
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
	return c, nil
}

func (c *Cluster) Join() error {
	_, err := c.memberList.Join(c.cfg.Hosts)
	return err
}

func (c *Cluster) Leave() error {
	return c.memberList.Leave(leaveTimeout)
}

func (c *Cluster) Send(stanza xmpp.Stanza, toNode string) error {
	return nil
}

func (c *Cluster) Shutdown() error {
	if c.memberList != nil {
		return c.memberList.Shutdown()
	}
	return nil
}

func (c *Cluster) handleNotifyJoin(n *memberlist.Node) {
	if c.delegate != nil {
		c.delegate.NodeJoined(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyLeave(n *memberlist.Node) {
	if c.delegate != nil {
		c.delegate.NodeLeft(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyUpdate(n *memberlist.Node) {
}

func (c *Cluster) handleNotifyMsg(msg []byte) {
}
