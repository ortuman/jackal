/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/xmpp/jid"
)

const clusterMailboxSize = 32768

const leaveTimeout = time.Second * 5

type Node struct {
	Name string
}

type Delegate interface {
	NotifyMessage(msg *Message)
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
	go c.loop()
	return c, nil
}

func (c *Cluster) Join() error {
	log.Infof("local node: %s", c.LocalNode())

	c.membersMu.Lock()
	for _, m := range c.memberList.Members() {
		if m.Name == c.LocalNode() {
			continue
		}
		log.Infof("registered cluster node: %s", m.Name)
		c.members[m.Name] = m
	}
	c.membersMu.Unlock()
	_, err := c.memberList.Join(c.cfg.Hosts)
	return err
}

func (c *Cluster) LocalNode() string {
	return c.cfg.Name
}

func (c *Cluster) C2SStream(identifier string, jid *jid.JID, node string) *C2S {
	return newC2S(identifier, jid, node, c)
}

func (c *Cluster) SendMessageTo(node string, msg *Message) {
	c.actorCh <- func() {
		c.membersMu.RLock()
		to := c.members[node]
		c.membersMu.RUnlock()
		if to == nil {
			log.Errorf("cannot send message: node %s not found", node)
			return
		}
		msgBytes := c.encodeMessage(msg)
		if err := c.memberList.SendReliable(to, msgBytes); err != nil {
			log.Error(err)
		}
	}
}

func (c *Cluster) BroadcastMessage(msg *Message) {
	c.actorCh <- func() {
		if err := c.broadcast(msg); err != nil {
			log.Error(err)
		}
	}
}

func (c *Cluster) Shutdown() error {
	errCh := make(chan error, 1)
	c.actorCh <- func() {
		defer close(c.actorCh)

		if err := c.memberList.Leave(leaveTimeout); err != nil {
			errCh <- err
			return
		}
		if err := c.memberList.Shutdown(); err != nil {
			errCh <- err
			return
		}
		close(errCh)
	}
	return <-errCh
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

func (c *Cluster) broadcast(msg *Message) error {
	msgBytes := c.encodeMessage(msg)
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
	if n.Name == c.LocalNode() {
		return
	}
	c.membersMu.Lock()
	c.members[n.Name] = n
	c.membersMu.Unlock()

	log.Infof("registered cluster node: %s", n.Name)
	if c.delegate != nil && n.Name != c.LocalNode() {
		c.delegate.NodeJoined(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyLeave(n *memberlist.Node) {
	if n.Name == c.LocalNode() {
		return
	}
	c.membersMu.Lock()
	delete(c.members, n.Name)
	c.membersMu.Unlock()

	log.Infof("unregistered cluster node: %s", n.Name)
	if c.delegate != nil && n.Name != c.LocalNode() {
		c.delegate.NodeLeft(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyUpdate(n *memberlist.Node) {
	if n.Name == c.LocalNode() {
		return
	}
	c.membersMu.Lock()
	c.members[n.Name] = n
	c.membersMu.Unlock()

	log.Infof("updated cluster node: %s", n.Name)
	if c.delegate != nil {
		c.delegate.NodeUpdated(&Node{Name: n.Name})
	}
}

func (c *Cluster) handleNotifyMsg(msg []byte) {
	if len(msg) == 0 {
		return
	}
	var m Message
	dec := gob.NewDecoder(bytes.NewReader(msg))
	if err := m.FromGob(dec); err != nil {
		log.Error(err)
		return
	}
	if c.delegate != nil {
		c.delegate.NotifyMessage(&m)
	}
}

func (c *Cluster) encodeMessage(msg *Message) []byte {
	buf := c.pool.Get()
	enc := gob.NewEncoder(buf)
	msg.ToGob(enc)
	msgBytes := make([]byte, buf.Len(), buf.Len())
	copy(msgBytes, buf.Bytes())
	c.pool.Put(buf)
	return msgBytes
}
