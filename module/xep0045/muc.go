/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
)

type Muc struct {
	cfg   *Config
	disco *xep0030.DiscoInfo
	reps  repository.Container
	// TODO maybe switch this into a list of strings (so that it doesn't get out of sync with
	// storage)
	allRooms []*mucmodel.Room
	router   router.Router
	runQueue *runqueue.RunQueue
	mu       sync.RWMutex
}

func New(cfg *Config, disco *xep0030.DiscoInfo, reps repository.Container, router router.Router) *Muc {
	// muc service needs a separate hostname
	if len(cfg.MucHost) == 0 || router.Hosts().IsLocalHost(cfg.MucHost) {
		log.Errorf("Muc service could not be started - invalid hostname")
		return nil
	}
	s := &Muc{
		cfg:      cfg,
		disco:    disco,
		reps:     reps,
		router:   router,
		runQueue: runqueue.New("muc"),
	}
	router.Hosts().AddMucHostname(cfg.MucHost)
	if disco != nil {
		setupDiscoService(cfg, disco, s)
	}
	return s
}

func (s *Muc) ProcessPresence(ctx context.Context, presence *xmpp.Presence) {
	s.runQueue.Run(func() {
		s.processPresence(ctx, presence)
	})
}

// TODO this function only handles room creation atm, it should probably be split into create/
// send a msg
func (s *Muc) processPresence(ctx context.Context, presence *xmpp.Presence) {
	from := presence.FromJID()
	to := presence.ToJID()
	roomJID := to.ToBareJID()
	nick := to.Resource()
	roomName := roomJID.Node()

	// TODO write all of the checks, return appropriate error codes if data is not valid

	// TODO there is an error here atm
	locked := false
	xEl := presence.Elements().ChildNamespace("x", mucNamespace)
	if xEl != nil && xEl.Text() == "" {
		locked = true
	}

	if locked {
		log.Infof("LOCKED")
	} else {
		log.Infof("NOT LOCKED")
	}

	err := s.newRoom(ctx, from, to, roomJID, roomName, nick, locked)
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("New room created, room JID is %s", roomJID.String())
	err = s.sendRoomCreateAck(ctx, to, from)
	if err != nil {
		log.Error(err)
	}
}

func (s *Muc) GetMucHostname() string {
	return s.cfg.MucHost
}

func (s *Muc) Shutdown() error {
	c := make(chan struct{})
	s.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}
