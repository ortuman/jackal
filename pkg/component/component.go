// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"errors"
	"fmt"
	"sync"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/pkg/hook"
)

// ErrComponentNotFound will be returned by ProcessStanza in case the receiver component is not registered.
var ErrComponentNotFound = errors.New("component: not found")

// Component represents generic component interface.
type Component interface {
	// Host returns component host address.
	Host() string

	// Name returns component name.
	Name() string

	// ProcessStanza will be called in case stanza is requested to processed by this component.
	ProcessStanza(ctx context.Context, stanza stravaganza.Stanza) error

	// Start starts component.
	Start(ctx context.Context) error

	// Stop stops component.
	Stop(ctx context.Context) error
}

// Components is the global component hub.
type Components struct {
	mtx    sync.RWMutex
	comps  map[string]Component
	hk     *hook.Hooks
	logger kitlog.Logger
}

// NewComponents returns a new initialized Components instance.
func NewComponents(
	components []Component,
	hk *hook.Hooks,
	logger kitlog.Logger,
) *Components {
	cs := &Components{
		comps:  make(map[string]Component),
		hk:     hk,
		logger: logger,
	}
	for _, comp := range components {
		cs.comps[comp.Host()] = comp
	}
	return cs
}

// RegisterComponent registers a new component.
func (c *Components) RegisterComponent(ctx context.Context, comp Component) error {
	if err := comp.Start(ctx); err != nil {
		return err
	}
	cHost := comp.Host()
	c.mtx.Lock()
	c.comps[cHost] = comp
	c.mtx.Unlock()

	return nil
}

// UnregisterComponent unregisters a previously registered component.
func (c *Components) UnregisterComponent(ctx context.Context, cHost string) error {
	c.mtx.RLock()
	comp := c.comps[cHost]
	c.mtx.RUnlock()
	if comp == nil {
		return fmt.Errorf("%w: %s", ErrComponentNotFound, cHost)
	}
	if err := comp.Stop(ctx); err != nil {
		return err
	}
	c.mtx.Lock()
	delete(c.comps, cHost)
	c.mtx.Unlock()

	return nil
}

// Component returns the component associated to cHost.
func (c *Components) Component(cHost string) Component {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.comps[cHost]
}

// AllComponents returns all registered components.
func (c *Components) AllComponents() []Component {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	ret := make([]Component, 0, len(c.comps))
	for _, comp := range c.comps {
		ret = append(ret, comp)
	}
	return ret
}

// IsComponentHost tells whether cHost corresponds to some registered component.
func (c *Components) IsComponentHost(cHost string) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.comps[cHost] != nil
}

// ProcessStanza will route stanza to proper component based on receiver JID address.
func (c *Components) ProcessStanza(ctx context.Context, stanza stravaganza.Stanza) error {
	cHost := stanza.ToJID().Domain()

	c.mtx.RLock()
	comp := c.comps[cHost]
	c.mtx.RUnlock()

	if comp == nil {
		return fmt.Errorf("%w: %s", ErrComponentNotFound, cHost)
	}
	return comp.ProcessStanza(ctx, stanza)
}

// Start starts components.
func (c *Components) Start(ctx context.Context) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// start components
	var hosts []string
	for _, comp := range c.comps {
		if err := comp.Start(ctx); err != nil {
			return err
		}
		hosts = append(hosts, comp.Host())
	}
	level.Info(c.logger).Log("msg", "started components", "total_components", len(c.comps))

	_, err := c.hk.Run(ctx, hook.ComponentsStarted, &hook.ExecutionContext{
		Info: &hook.ComponentsInfo{
			Hosts: hosts,
		},
		Sender: c,
	})
	return err
}

// Stop stops components.
func (c *Components) Stop(ctx context.Context) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// stop components
	var hosts []string
	for _, comp := range c.comps {
		if err := comp.Stop(ctx); err != nil {
			return err
		}
		hosts = append(hosts, comp.Host())
	}
	level.Info(c.logger).Log("msg", "stopped components", "total_components", len(c.comps))

	_, err := c.hk.Run(ctx, hook.ComponentsStopped, &hook.ExecutionContext{
		Info: &hook.ComponentsInfo{
			Hosts: hosts,
		},
		Sender: c,
	})
	return err
}
