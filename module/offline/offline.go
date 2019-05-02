/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package offline

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

const offlineNamespace = "msgoffline"

const offlineDeliveredCtxKey = "offline:delivered"

// Offline represents an offline server stream module.
type Offline struct {
	cfg      *Config
	router   *router.Router
	runQueue *runqueue.RunQueue
}

// New returns an offline server stream module.
func New(config *Config, disco *xep0030.DiscoInfo, router *router.Router) *Offline {
	r := &Offline{
		cfg:      config,
		router:   router,
		runQueue: runqueue.New("xep0030"),
	}
	if disco != nil {
		disco.RegisterServerFeature(offlineNamespace)
	}
	return r
}

// ArchiveMessage archives a new offline messages into the storage.
func (x *Offline) ArchiveMessage(message *xmpp.Message) {
	x.runQueue.Post(func() { x.archiveMessage(message) })
}

// DeliverOfflineMessages delivers every archived offline messages to the peer
// deleting them from storage.
func (x *Offline) DeliverOfflineMessages(stm stream.C2S) {
	x.runQueue.Post(func() { x.deliverOfflineMessages(stm) })
}

// Shutdown shuts down offline module.
func (x *Offline) Shutdown() error {
	x.runQueue.Stop()
	return nil
}

func (x *Offline) archiveMessage(message *xmpp.Message) {
	if !isMessageArchivable(message) {
		return
	}
	toJID := message.ToJID()
	queueSize, err := storage.CountOfflineMessages(toJID.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if queueSize >= x.cfg.QueueSize {
		x.router.Route(message.ServiceUnavailableError())
		return
	}
	delayed, _ := xmpp.NewMessageFromElement(message, message.FromJID(), message.ToJID())
	delayed.Delay(message.FromJID().Domain(), "Offline Storage")
	if err := storage.InsertOfflineMessage(delayed, toJID.Node()); err != nil {
		log.Error(err)
		x.router.Route(message.InternalServerError())
		return
	}
	log.Infof("archived offline message... id: %s", message.ID())

	if x.cfg.Gateway != nil {
		if err := x.cfg.Gateway.Route(message); err != nil {
			log.Errorf("bad offline gateway: %v", err)
		}
	}
}

func (x *Offline) deliverOfflineMessages(stm stream.C2S) {
	if stm.GetBool(offlineDeliveredCtxKey) {
		return // already delivered
	}
	// deliver offline messages
	userJID := stm.JID()
	messages, err := storage.FetchOfflineMessages(userJID.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if len(messages) == 0 {
		return
	}
	log.Infof("delivering offline messages: %s... count: %d", userJID, len(messages))

	for _, m := range messages {
		x.router.Route(m)
	}
	if err := storage.DeleteOfflineMessages(userJID.Node()); err != nil {
		log.Error(err)
	}
	stm.SetBool(offlineDeliveredCtxKey, true)
}

func isMessageArchivable(message *xmpp.Message) bool {
	return message.IsNormal() || (message.IsChat() && message.IsMessageWithBody())
}
