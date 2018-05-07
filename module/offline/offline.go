/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package offline

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const offlineNamespace = "msgoffline"

// Config represents Offline Storage module configuration.
type Config struct {
	QueueSize int `yaml:"queue_size"`
}

// ModOffline represents an offline server stream module.
type ModOffline struct {
	cfg     *Config
	stm     c2s.Stream
	actorCh chan func()
	doneCh  chan struct{}
}

// New returns an offline server stream module.
func New(config *Config, stm c2s.Stream) *ModOffline {
	r := &ModOffline{
		cfg:     config,
		stm:     stm,
		actorCh: make(chan func(), 32),
		doneCh:  make(chan struct{}),
	}
	go r.actorLoop()
	return r
}

// AssociatedNamespaces returns namespaces associated
// with offline module.
func (o *ModOffline) AssociatedNamespaces() []string {
	return []string{offlineNamespace}
}

// Done signals stream termination.
func (o *ModOffline) Done() {
	o.doneCh <- struct{}{}
}

// ArchiveMessage archives a new offline messages into the storage.
func (o *ModOffline) ArchiveMessage(message *xml.Message) {
	o.actorCh <- func() {
		o.archiveMessage(message)
	}
}

// DeliverOfflineMessages delivers every archived offline messages to the peer
// deleting them from storage.
func (o *ModOffline) DeliverOfflineMessages() {
	o.actorCh <- func() {
		o.deliverOfflineMessages()
	}
}

func (o *ModOffline) actorLoop() {
	for {
		select {
		case f := <-o.actorCh:
			f()
		case <-o.doneCh:
			return
		}
	}
}

func (o *ModOffline) archiveMessage(message *xml.Message) {
	toJid := message.ToJID()
	queueSize, err := storage.Instance().CountOfflineMessages(toJid.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if queueSize >= o.cfg.QueueSize {
		response := xml.NewElementFromElement(message)
		response.SetFrom(toJid.String())
		response.SetTo(o.stm.JID().String())
		o.stm.SendElement(response.ServiceUnavailableError())
		return
	}
	delayed := xml.NewElementFromElement(message)
	delayed.Delay(o.stm.Domain(), "Offline Storage")
	if err := storage.Instance().InsertOfflineMessage(delayed, toJid.Node()); err != nil {
		log.Errorf("%v", err)
		return
	}
	log.Infof("archived offline message... id: %s", message.ID())
}

func (o *ModOffline) deliverOfflineMessages() {
	messages, err := storage.Instance().FetchOfflineMessages(o.stm.Username())
	if err != nil {
		log.Error(err)
		return
	}
	if len(messages) == 0 {
		return
	}
	log.Infof("delivering offline messages... count: %d", len(messages))

	for _, m := range messages {
		o.stm.SendElement(m)
	}
	if err := storage.Instance().DeleteOfflineMessages(o.stm.Username()); err != nil {
		log.Error(err)
	}
}
