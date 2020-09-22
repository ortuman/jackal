/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"
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

func (s *Muc) GetRoomAdmins(ctx context.Context, r *mucmodel.Room) []string {
	admins := make([]string, 0)
	for _, occJID := range r.UserToOccupant {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			log.Error(err)
			return nil
		}
		if o.IsAdmin() {
			admins = append(admins, occJID.String())
		}
	}
	return admins
}

func (s *Muc) GetRoomOwners(ctx context.Context, r *mucmodel.Room) []string {
	owners := make([]string, 0)
	for bareJID, occJID := range r.UserToOccupant {
		o, err := s.repOccupant.FetchOccupant(ctx, &occJID)
		if err != nil {
			log.Error(err)
			return nil
		}
		if o.IsOwner() {
			owners = append(owners, bareJID.String())
		}
	}
	return owners
}

func (s *Muc) SetRoomAdmin(ctx context.Context, room *mucmodel.Room, adminJID *jid.JID) error {
	// check if the occupant is in the room
	occJID, found := room.UserToOccupant[*adminJID]
	if !found {
		return fmt.Errorf("muc: user has to enter the room before it can be made admin")
	}

	occupant, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		return err
	}

	err = occupant.SetAffiliation("admin")
	if err != nil {
		return err
	}

	err = s.repOccupant.UpsertOccupant(ctx, occupant)
	if err != nil {
		return err
	}

	return nil
}

func (s *Muc) SetRoomOwner(ctx context.Context, room *mucmodel.Room, ownerJID *jid.JID) error {
	// check if the occupant is in the room
	occJID, found := room.UserToOccupant[*ownerJID]
	if !found {
		return fmt.Errorf("muc: user has to enter the room before it can be made owner")
	}

	occupant, err := s.repOccupant.FetchOccupant(ctx, &occJID)
	if err != nil {
		return err
	}

	err = occupant.SetAffiliation("owner")
	if err != nil {
		return err
	}

	err = s.repOccupant.UpsertOccupant(ctx, occupant)
	if err != nil {
		return err
	}

	return nil
}

func (s *Muc) AddOccupantToRoom(ctx context.Context, room *mucmodel.Room, occupant *mucmodel.Occupant) error {
	room.AddOccupant(occupant)
	return s.repRoom.UpsertRoom(ctx, room)
}
