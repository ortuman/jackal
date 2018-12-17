/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package offline

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 2048

const offlineNamespace = "msgoffline"

const offlineDeliveredCtxKey = "offline:delivered"

// Config represents Offline Storage module configuration.
type Config struct {
	QueueSize int `yaml:"queue_size"`
}

// Offline represents an offline server stream module.
type Offline struct {
	cfg        *Config
	router     *router.Router
	actorCh    chan func()
	shutdownCh chan chan bool
}

// New returns an offline server stream module.
func New(config *Config, disco *xep0030.DiscoInfo, router *router.Router) (*Offline, chan<- chan bool) {
	r := &Offline{
		cfg:        config,
		router:     router,
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan bool),
	}
	go r.loop()
	if disco != nil {
		disco.RegisterServerFeature(offlineNamespace)
	}
	return r, r.shutdownCh
}

// ArchiveMessage archives a new offline messages into the storage.
func (o *Offline) ArchiveMessage(message *xmpp.Message) {
	o.actorCh <- func() { o.archiveMessage(message) }
}

// DeliverOfflineMessages delivers every archived offline messages to the peer
// deleting them from storage.
func (o *Offline) DeliverOfflineMessages(stm stream.C2S) {
	o.actorCh <- func() { o.deliverOfflineMessages(stm) }
}

// runs on it's own goroutine
func (o *Offline) loop() {
	for {
		select {
		case f := <-o.actorCh:
			f()
		case c := <-o.shutdownCh:
			c <- true
			return
		}
	}
}

func (o *Offline) archiveMessage(message *xmpp.Message) {
	if !isMessageArchivable(message) {
		return
	}
	toJID := message.ToJID()
	queueSize, err := storage.CountOfflineMessages(toJID.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if queueSize >= o.cfg.QueueSize {
		o.router.Route(message.ServiceUnavailableError())
		return
	}
	delayed, _ := xmpp.NewMessageFromElement(message, message.FromJID(), message.ToJID())
	delayed.Delay(message.FromJID().Domain(), "Offline Storage")
	if err := storage.InsertOfflineMessage(delayed, toJID.Node()); err != nil {
		log.Error(err)
		o.router.Route(message.InternalServerError())
		return
	}
	log.Infof("archived offline message... id: %s", message.ID())
}

func (o *Offline) deliverOfflineMessages(stm stream.C2S) {
	if stm.GetBool(offlineDeliveredCtxKey) {
		return // already delivered
	}
	// deliver offline messages
	userJID := stm.JID()
	msgs, err := storage.FetchOfflineMessages(userJID.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if len(msgs) == 0 {
		return
	}
	log.Infof("delivering offline msgs: %s... count: %d", userJID, len(msgs))

	for _, m := range msgs {
		o.router.Route(m)
	}
	if err := storage.DeleteOfflineMessages(userJID.Node()); err != nil {
		log.Error(err)
	}
	stm.SetBool(offlineDeliveredCtxKey, true)
}

func isMessageArchivable(message *xmpp.Message) bool {
	return message.IsNormal() || (message.IsChat() && message.IsMessageWithBody())
}
