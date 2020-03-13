/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package offline

import (
	"context"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
)

const offlineNamespace = "msgoffline"

const hintsNamespace = "urn:xmpp:hints"

const offlineDeliveredCtxKey = "offline:delivered"

// Offline represents an offline server stream module.
type Offline struct {
	cfg        *Config
	runQueue   *runqueue.RunQueue
	router     router.Router
	offlineRep repository.Offline
}

// New returns an offline server stream module.
func New(config *Config, disco *xep0030.DiscoInfo, router router.Router, offlineRep repository.Offline) *Offline {
	r := &Offline{
		cfg:        config,
		runQueue:   runqueue.New("xep0030"),
		router:     router,
		offlineRep: offlineRep,
	}
	if disco != nil {
		disco.RegisterServerFeature(offlineNamespace)
	}
	return r
}

// ArchiveMessage archives a new offline messages into the storage.
func (x *Offline) ArchiveMessage(ctx context.Context, message *xmpp.Message) {
	x.runQueue.Run(func() { x.archiveMessage(ctx, message) })
}

// DeliverOfflineMessages delivers every archived offline messages to the peer
// deleting them from storage.
func (x *Offline) DeliverOfflineMessages(ctx context.Context, stm stream.C2S) {
	x.runQueue.Run(func() { x.deliverOfflineMessages(ctx, stm) })
}

// Shutdown shuts down offline module.
func (x *Offline) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *Offline) archiveMessage(ctx context.Context, message *xmpp.Message) {
	if !isMessageArchivable(message) {
		return
	}
	toJID := message.ToJID()
	queueSize, err := x.offlineRep.CountOfflineMessages(ctx, toJID.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if queueSize >= x.cfg.QueueSize {
		_ = x.router.Route(ctx, message.ServiceUnavailableError())
		return
	}
	delayed, _ := xmpp.NewMessageFromElement(message, message.FromJID(), message.ToJID())
	delayed.Delay(message.FromJID().Domain(), "Offline Storage")
	if err := x.offlineRep.InsertOfflineMessage(ctx, delayed, toJID.Node()); err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, message.InternalServerError())
		return
	}
	log.Infof("archived offline message... id: %s", message.ID())

	if x.cfg.Gateway != nil {
		if err := x.cfg.Gateway.Route(message); err != nil {
			log.Errorf("bad offline gateway: %v", err)
		}
	}
}

func (x *Offline) deliverOfflineMessages(ctx context.Context, stm stream.C2S) {
	delivered, _ := stm.Value(offlineDeliveredCtxKey).(bool)
	if delivered {
		return // already delivered
	}
	// deliver offline messages
	userJID := stm.JID()
	messages, err := x.offlineRep.FetchOfflineMessages(ctx, userJID.Node())
	if err != nil {
		log.Error(err)
		return
	}
	if len(messages) == 0 {
		return
	}
	log.Infof("delivering offline messages: %s... count: %d", userJID, len(messages))

	for i := 0; i < len(messages); i++ {
		_ = x.router.Route(ctx, &messages[i])
	}
	if err := x.offlineRep.DeleteOfflineMessages(ctx, userJID.Node()); err != nil {
		log.Error(err)
	}
	stm.SetValue(offlineDeliveredCtxKey, true)
}

func isMessageArchivable(message *xmpp.Message) bool {
	if message.Elements().ChildNamespace("no-store", hintsNamespace) != nil {
		return false
	}
	if message.Elements().ChildNamespace("store", hintsNamespace) != nil {
		return true
	}
	return message.IsNormal() || (message.IsChat() && message.IsMessageWithBody())
}
