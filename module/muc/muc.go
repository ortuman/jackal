/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package muc

import (
	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
)

// TODO change the name of this type, Service is a terrible name
type Service struct {
	cfg         *Config
	disco       *xep0030.DiscoInfo
	reps        repository.Container
	publicRooms []*mucmodel.Room
}

func New(cfg *Config, disco *xep0030.DiscoInfo, reps repository.Container, router router.Router) *Service {
	// muc service needs a separate hostname
	if len(cfg.MucHost) == 0 || router.Hosts().IsLocalHost(cfg.MucHost) {
		log.Errorf("Muc service could not be started - invalid hostname")
		return nil
	}
	s := &Service{
		cfg:   cfg,
		disco: disco,
		reps:  reps,
	}
	router.Hosts().AddMucHostname(cfg.MucHost)
	if disco != nil {
		setupDiscoMuc(cfg, disco, s)
	}
	return s
}

func (s *Service) GetMucHostname() string {
	return s.cfg.MucHost
}

func (s *Service) Shutdown() error {
	// TODO for some reason every module has a runqueue, and it needs to be closed here, figure out
	// why this is the case
	return nil
}
