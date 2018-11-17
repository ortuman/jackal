/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"time"

	"github.com/ortuman/jackal/log"

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

type Cluster interface {
	Enabled() bool
	Join() error
	Leave() error
	Send(stanza xmpp.Stanza, toNode string) error
	Shutdown() error
}

type cluster struct {
	cfg        *Config
	delegate   Delegate
	memberList *memberlist.Memberlist
}

func New(config *Config, delegate Delegate) (Cluster, error) {
	c := &cluster{}
	if config == nil {
		return c, nil
	}
	c.cfg = config
	c.delegate = delegate
	conf := memberlist.DefaultLocalConfig()
	conf.Name = config.Name
	conf.BindPort = config.BindPort
	conf.Delegate = &memberListDelegate{cluster: c}
	conf.Events = &memberListEventDelegate{cluster: c}
	ml, err := memberlist.Create(conf)
	if err != nil {
		return nil, err
	}
	c.memberList = ml
	return c, nil
}

// Enabled returns whether or not cluster has been enabled.
func (c *cluster) Enabled() bool {
	return c.memberList != nil
}

func (c *cluster) SetDelegate(delegate Delegate) {
	c.delegate = delegate
}

func (c *cluster) Join() error {
	_, err := c.memberList.Join(c.cfg.Hosts)
	return err
}

func (c *cluster) Leave() error {
	return c.memberList.Leave(leaveTimeout)
}

func (c *cluster) Send(stanza xmpp.Stanza, toNode string) error {
	return nil
}

func (c *cluster) Shutdown() error {
	if c.memberList != nil {
		return c.memberList.Shutdown()
	}
	return nil
}

func (c *cluster) handleNotifyJoin(n *memberlist.Node) {
	log.Infof("join notified: %s", n.Name)
	if c.delegate != nil {
		c.delegate.NodeJoined(&Node{Name: n.Name})
	}
}

func (c *cluster) handleNotifyLeave(n *memberlist.Node) {
	if c.delegate != nil {
		c.delegate.NodeLeft(&Node{Name: n.Name})
	}
}

func (c *cluster) handleNotifyUpdate(n *memberlist.Node) {
	log.Infof("update notified: %s", n.Name)
}

func (c *cluster) handleNotifyMsg(msg []byte) {
}
