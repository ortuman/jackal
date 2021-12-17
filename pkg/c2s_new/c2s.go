package c2s_new

import (
	"context"

	"github.com/ortuman/jackal/pkg/auth/pepper"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type C2S struct {
	listeners []*socketListener
}

func New(
	cfg Config,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	resMng *ResourceManager,
	rep repository.Repository,
	peppers *pepper.Keys,
	shapers shaper.Shapers,
	hk *hook.Hooks,
) *C2S {
	var c C2S
	for _, lnCfg := range cfg.Listeners {
		ln := newSocketListener(
			lnCfg,
			hosts,
			router,
			comps,
			mods,
			resMng,
			rep,
			peppers,
			shapers,
			hk,
		)
		c.listeners = append(c.listeners, ln)
	}
	return &c
}

func (c *C2S) Start(ctx context.Context) error {
	for _, ln := range c.listeners {
		if err := ln.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *C2S) Stop(ctx context.Context) error {
	for _, ln := range c.listeners {
		if err := ln.Stop(ctx); err != nil {
			return err
		}
	}
	return nil
}
