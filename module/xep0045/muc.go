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
	"github.com/ortuman/jackal/xmpp/jid"
)

type Muc struct {
	cfg         *Config
	disco       *xep0030.DiscoInfo
	repRoom     repository.Room
	repOccupant repository.Occupant
	// room JIDs of all rooms on the service
	allRooms []jid.JID
	router   router.Router
	runQueue *runqueue.RunQueue
	mu       sync.RWMutex
}

func New(cfg *Config, disco *xep0030.DiscoInfo, router router.Router, repRoom repository.Room,
	repOccupant repository.Occupant) *Muc {
	// muc service needs a separate hostname
	if len(cfg.MucHost) == 0 || router.Hosts().IsLocalHost(cfg.MucHost) {
		log.Errorf("Muc service could not be started - invalid hostname")
		return nil
	}
	s := &Muc{
		cfg:         cfg,
		disco:       disco,
		repRoom:     repRoom,
		repOccupant: repOccupant,
		router:      router,
		runQueue:    runqueue.New("muc"),
	}
	router.Hosts().AddMucHostname(cfg.MucHost)
	if disco != nil {
		setupDiscoService(cfg, disco, s)
	}
	return s
}

// accepting all IQs aimed at the conference service
func (s *Muc) MatchesIQ(iq *xmpp.IQ) bool {
	return s.router.Hosts().IsConferenceHost(iq.ToJID().Domain())
}

func (s *Muc) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	s.runQueue.Run(func() {
		s.processIQ(ctx, iq)
	})
}

func (s *Muc) processIQ(ctx context.Context, iq *xmpp.IQ) {
	roomJID := iq.ToJID().ToBareJID()
	room, err := s.repRoom.FetchRoom(ctx, roomJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
		return
	}
	if room == nil {
		_ = s.router.Route(ctx, iq.ItemNotFoundError())
		return
	}

	switch {
	case isIQForInstantRoomCreate(iq):
		s.createInstantRoom(ctx, room, iq)
	case isIQForRoomConfigRequest(iq):
		s.sendRoomConfiguration(ctx, room, iq)
	case isIQForRoomConfigSubmission(iq):
		s.processRoomConfiguration(ctx, room, iq)
	default:
		_ = s.router.Route(ctx, iq.BadRequestError())
	}

}

func (s *Muc) ProcessPresence(ctx context.Context, presence *xmpp.Presence) {
	s.runQueue.Run(func() {
		s.processPresence(ctx, presence)
	})
}

func (s *Muc) processPresence(ctx context.Context, presence *xmpp.Presence) {
	roomJID := presence.ToJID().ToBareJID()
	room, err := s.repRoom.FetchRoom(ctx, roomJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, presence.InternalServerError())
		return
	}

	switch {
	case isPresenceToEnterRoom(presence):
		s.enterRoom(ctx, room, presence)
	default:
		_ = s.router.Route(ctx, presence.BadRequestError())
	}
}

func (s *Muc) GetMucHostname() string {
	return s.cfg.MucHost
}

func (s *Muc) GetDefaultRoomConfig() *mucmodel.RoomConfig {
	conf := s.cfg.RoomDefaults
	return &conf
}

func (s *Muc) Shutdown() error {
	c := make(chan struct{})
	s.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}
