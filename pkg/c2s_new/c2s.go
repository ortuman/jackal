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
	"golang.org/x/sync/errgroup"
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
	eGroup, egCtx := errgroup.WithContext(ctx)
	for i := 0; i < len(c.listeners); i++ {
		idx := i
		eGroup.Go(func() error {
			ln := c.listeners[idx]
			return ln.Start(egCtx)
		})
	}
	return eGroup.Wait()
}

func (c *C2S) Stop(ctx context.Context) error {
	eGroup, egCtx := errgroup.WithContext(ctx)
	for i := 0; i < len(c.listeners); i++ {
		idx := i
		eGroup.Go(func() error {
			ln := c.listeners[idx]
			return ln.Stop(egCtx)
		})
	}
	return eGroup.Wait()
}
