/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"sync"
	"time"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xml"
)

type Offline struct {
	queue concurrent.OperationQueue
	cfg   *config.ModOffline
	strm  Stream
	once  sync.Once
}

func NewOffline(config *config.ModOffline, stream Stream) *Offline {
	return &Offline{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second,
		},
		cfg:  config,
		strm: stream,
	}
}

func (o *Offline) AssociatedNamespaces() []string {
	return []string{"msgoffline"}
}

func (o *Offline) ArchiveMessage(message *xml.Message) {
	o.queue.Async(func() { o.archiveMessage(message) })
}

func (o *Offline) DeliverOfflineMessages() {
	o.once.Do(func() {
		o.queue.Async(func() { o.deliverOfflineMessages() })
	})
}

func (o *Offline) archiveMessage(message *xml.Message) {
	toJid := message.ToJID()
	queueSize, err := storage.Instance().CountOfflineMessages(toJid.Node())
	if err != nil {
		log.Error(err)
		return
	}
	exists, err := storage.Instance().UserExists(toJid.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if !exists || queueSize >= o.cfg.QueueSize {
		response := message.MutableCopy()
		response.SetFrom(toJid.String())
		response.SetTo(o.strm.JID().String())
		o.strm.SendElement(response.ServiceUnavailableError())
		return
	}
	delayed := message.Delayed(o.strm.Domain(), "Offline Storage")
	if err := storage.Instance().InsertOfflineMessage(delayed, toJid.Node()); err != nil {
		log.Errorf("%v", err)
		return
	}
	log.Infof("archived offline message... id: %s", message.ID())
}

func (o *Offline) deliverOfflineMessages() {
	messages, err := storage.Instance().FetchOfflineMessages(o.strm.Username())
	if err != nil {
		log.Error(err)
		return
	}
	if len(messages) == 0 {
		return
	}
	log.Infof("delivering offline messages... count: %d", len(messages))

	for _, m := range messages {
		o.strm.SendElement(m)
	}
	if err := storage.Instance().DeleteOfflineMessages(o.strm.Username()); err != nil {
		log.Error(err)
	}
}
