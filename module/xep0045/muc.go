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
	cfg   *Config
	disco *xep0030.DiscoInfo
	reps  repository.Container
	// room JIDs of all rooms on the service
	allRooms []*jid.JID
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
	roomJID := iq.ToJID()
	room, err := s.reps.Room().FetchRoom(ctx, roomJID)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
		return
	}
	if room == nil {
		_ = s.router.Route(ctx, iq.BadRequestError())
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

// TODO this function only handles room creation atm, it should probably be split into create/
// send a msg
func (s *Muc) processPresence(ctx context.Context, presence *xmpp.Presence) {
	from := presence.FromJID()
	to := presence.ToJID()
	nick := to.Resource()
	roomName := to.Node()

	// TODO once all presence handling is written, write all of the checks, return
	// appropriate error codes if data is not valid
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

	err := s.newRoom(ctx, from, to, roomName, nick, locked)
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("New room created, room JID is %s", to.ToBareJID().String())
	err = s.sendRoomCreateAck(ctx, to, from)
	if err != nil {
		log.Error(err)
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

func (s *Muc) GetRoomAdmins(ctx context.Context, r *mucmodel.Room) (admins []string) {
	for _, occJID := range r.UserToOccupant {
		o, err := s.GetOccupant(ctx, &occJID)
		if err != nil {
			log.Error(err)
			return nil
		}
		if o.IsAdmin() {
			admins = append(admins, occJID.String())
		}
	}
	return
}

func (s *Muc) GetRoomOwners(ctx context.Context, r *mucmodel.Room) (owners []string) {
	for _, occJID := range r.UserToOccupant {
		o, err := s.GetOccupant(ctx, &occJID)
		if err != nil {
			log.Error(err)
			return nil
		}
		if o.IsOwner() {
			owners = append(owners, occJID.String())
		}
	}
	return
}

func (s *Muc) GetOccupant(ctx context.Context, occJID *jid.JID) (*mucmodel.Occupant, error) {
	return s.reps.Occupant().FetchOccupant(ctx, occJID)
}

func (s *Muc) GetRoom(ctx context.Context, roomJID *jid.JID) (*mucmodel.Room, error) {
	return s.reps.Room().FetchRoom(ctx, roomJID)
}
