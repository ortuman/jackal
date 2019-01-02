/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package component

import (
	"context"
	"fmt"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

// Component represents a generic component interface.
type Component interface {
	Host() string
	ProcessStanza(stanza xmpp.Stanza, stm stream.C2S)
}

// Components represents a set of preconfigured components.
type Components struct {
	comps       map[string]Component
	shutdownChs []chan<- chan bool
}

// New returns a set of components derived from a concrete configuration.
func New(config *Config, discoInfo *xep0030.DiscoInfo) *Components {
	comps := &Components{
		comps: make(map[string]Component),
	}
	cs, shutdownChs := loadComponents(config, discoInfo)
	for _, c := range cs {
		host := c.Host()
		if _, ok := comps.comps[host]; ok {
			log.Fatal(fmt.Errorf("component host name conflict: %s", host))
		}
		comps.comps[host] = c
	}
	comps.shutdownChs = shutdownChs
	return comps
}

// Get returns a specific component associated to host name.
func (cs *Components) Get(host string) Component {
	return cs.comps[host]
}

// GetAll returns all initialized components.
func (cs *Components) GetAll() []Component {
	var ret []Component
	for _, comp := range cs.comps {
		ret = append(ret, comp)
	}
	return ret
}

// Shutdown gracefully shuts down components instance.
func (cs *Components) Shutdown(ctx context.Context) error {
	select {
	case <-cs.shutdown():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (cs *Components) shutdown() <-chan bool {
	c := make(chan bool)
	go func() {
		// shutdown components in reverse order
		for i := len(cs.shutdownChs) - 1; i >= 0; i-- {
			shutdownCh := cs.shutdownChs[i]
			wc := make(chan bool, 1)
			shutdownCh <- wc
			<-wc
		}
		close(c)
	}()
	return c
}

func loadComponents(_ *Config, _ *xep0030.DiscoInfo) ([]Component, []chan<- chan bool) {
	var comps []Component
	var shutdownChs []chan<- chan bool
	/*
		discoInfo := module.Modules().DiscoInfo
		if cfg.HttpUpload != nil {
			comp, shutdownCh := httpupload.New(cfg.HttpUpload, discoInfo)
			comps = append(comps, comp)
			shutdownChs = append(shutdownChs, shutdownCh)
		}
	*/
	return comps, shutdownChs
}
