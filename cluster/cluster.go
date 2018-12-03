/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"encoding/gob"
	"io/ioutil"
	"sync"
	"time"

	"github.com/ortuman/jackal/log"

	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const clusterMailboxSize = 4096

const leaveTimeout = time.Second * 5

type Node struct {
	Name string
}

type Delegate interface {
	NotifyMessage(interface{})
	NodeJoined(node *Node)
	NodeUpdated(node *Node)
	NodeLeft(node *Node)
}

type Cluster struct {
	cfg        *Config
	pool       *pool.BufferPool
	delegate   Delegate
	memberList *memberlist.Memberlist
	membersMu  sync.RWMutex
	members    map[string]*memberlist.Node
	actorCh    chan func()
}

func New(config *Config, delegate Delegate) (*Cluster, error) {
	if config == nil {
		return nil, nil
	}
	c := &Cluster{
		pool:    pool.NewBufferPool(),
		members: make(map[string]*memberlist.Node),
		actorCh: make(chan func(), clusterMailboxSize),
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
	return c, nil
}

func (c *Cluster) Join() error {
	_, err := c.memberList.Join(c.cfg.Hosts)
	return err
}

func (c *Cluster) LocalNode() string {
	return c.memberList.LocalNode().Name
}

func (c *Cluster) C2SStream(identifier string, jid *jid.JID, node string) *C2S {
	return newC2S(identifier, jid, node, c)
}

func (c *Cluster) BroadcastBindMessage(j *jid.JID) {
	c.actorCh <- func() {
		msg := &Message{
			Type: MsgBindType,
			Node: c.LocalNode(),
			JIDs: []*jid.JID{j},
		}
		err := c.broadcast(msg)
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *Cluster) BroadcastUnbindMessage(j *jid.JID) {
	c.actorCh <- func() {
		msg := &Message{
			Type: MsgUnbindType,
			Node: c.LocalNode(),
			JIDs: []*jid.JID{j},
		}
		err := c.broadcast(msg)
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *Cluster) BroadcastUpdatePresenceMessage(j *jid.JID, presence *xmpp.Presence) {
	c.actorCh <- func() {
		msg := &Message{
			Type: MsgUpdatePresenceType,
			Node: c.LocalNode(),
			JIDs: []*jid.JID{j},
		}
		err := c.broadcast(msg)
		if err != nil {
			log.Error(err)
		}
	}
}

func (c *Cluster) Send(msg []byte, toNode string) {
	c.actorCh <- func() {
		if err := c.send(msg, toNode); err != nil {
			log.Error(err)
		}
	}
}

func (c *Cluster) Shutdown() error {
	if c.memberList != nil {
		if err := c.memberList.Leave(leaveTimeout); err != nil {
			return err
		}
		if err := c.memberList.Shutdown(); err != nil {
			return err
		}
		close(c.actorCh)
	}
	return nil
}

func (c *Cluster) loop() {
	for f := range c.actorCh {
		f()
	}
}

func (c *Cluster) send(msg []byte, toNode string) error {
	c.membersMu.RLock()
	node := c.members[toNode]
	c.membersMu.RUnlock()
	if node == nil {
		return nil
	}
	return c.memberList.SendReliable(node, msg)
}

func (c *Cluster) broadcast(msg model.GobSerializer) error {
	buf := c.pool.Get()
	defer c.pool.Put(buf)

	enc := gob.NewEncoder(buf)
	msg.ToGob(enc)

	msgBytes := make([]byte, buf.Len(), buf.Len())
	copy(msgBytes, buf.Bytes())

	c.membersMu.RLock()
	defer c.membersMu.RUnlock()
	for _, node := range c.members {
		if err := c.memberList.SendReliable(node, msgBytes); err != nil {
			return err
		}
	}
	return nil
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

func (c *Cluster) handleNotifyMsg(msg interface{}) {
	if c.delegate != nil {
		c.delegate.NotifyMessage(msg)
	}
}
