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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const clusterMailboxSize = 32768

const leaveTimeout = time.Second * 5

type Metadata struct {
	Version   string
	GoVersion string
}

type Node struct {
	Name     string
	Metadata Metadata
}

type Delegate interface {
	NodeJoined(node *Node)
	NodeLeft(node *Node)

	NotifyMessage(msg *Message)
}

type Cluster struct {
	cfg        *Config
	buf        *bytes.Buffer
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
		buf:     bytes.NewBuffer(nil),
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

func (c *Cluster) C2SStream(jid *jid.JID, presence *xmpp.Presence, context map[string]interface{}, node string) *C2S {
	return newC2S(uuid.New().String(), jid, presence, context, node, c)
}

func (c *Cluster) SendMessageTo(node string, msg *Message) {
	c.actorCh <- func() {
		if err := c.send(msg, node); err != nil {
			log.Error(err)
			return
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

func (c *Cluster) send(msg *Message, toNode string) error {
	c.membersMu.RLock()
	node := c.members[toNode]
	c.membersMu.RUnlock()
	if node == nil {
		return fmt.Errorf("cannot send message: node %s not found", toNode)
	}
	return c.memberList.SendReliable(node, c.encodeMessage(msg))
}

func (c *Cluster) broadcast(msg *Message) error {
	msgBytes := c.encodeMessage(msg)
	c.membersMu.RLock()
	defer c.membersMu.RUnlock()
	for _, node := range c.members {
		if node.Name == c.LocalNode() {
			continue
		}
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
	var m Metadata
	if err := gob.NewDecoder(bytes.NewBuffer(n.Meta)).Decode(&m); err != nil {
		log.Warnf("%s", err)
		return
	}
	c.membersMu.Lock()
	c.members[n.Name] = n
	c.membersMu.Unlock()

	log.Infof("registered cluster node: %s", n.Name)
	if c.delegate != nil && n.Name != c.LocalNode() {
		c.delegate.NodeJoined(&Node{
			Name:     n.Name,
			Metadata: m,
		})
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
	defer c.buf.Reset()
	enc := gob.NewEncoder(c.buf)
	msg.ToGob(enc)
	msgBytes := make([]byte, c.buf.Len(), c.buf.Len())
	copy(msgBytes, c.buf.Bytes())
	return msgBytes
}
