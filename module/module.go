// Copyright 2020 The jackal Authors
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

package module

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
)

// Module represents generic module interface.
type Module interface {
	// Name returns specific module name.
	Name() string

	// StreamFeature returns module stream feature element.
	StreamFeature() stravaganza.Element

	// ServerFeatures returns module server features.
	ServerFeatures() []string

	// ServerFeatures returns module account features.
	AccountFeatures() []string

	// Start starts module.
	Start(ctx context.Context) error

	// Stop stops module.
	Stop(ctx context.Context) error
}

// EventHandler represents a event handler module type.
type EventHandler interface {
	Module
}

// IQHandler represents an iq handler module type.
type IQHandler interface {
	Module

	// MatchesNamespace tells whether iq child namespace corresponds to this module.
	MatchesNamespace(namespace string) bool

	// ProcessIQ will be invoked whenever iq stanza should be processed by this module.
	ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error
}

// Modules is the global module hub.
type Modules struct {
	iqHandlers    []IQHandler
	eventHandlers []EventHandler
	hosts         hosts
	router        router.Router
}

// NewModules returns a new initialized Modules instance.
func NewModules(
	iqHandlers []IQHandler,
	eventHandlers []EventHandler,
	hosts *host.Hosts,
	router router.Router,
) *Modules {
	return &Modules{
		iqHandlers:    iqHandlers,
		eventHandlers: eventHandlers,
		hosts:         hosts,
		router:        router,
	}
}

// Start starts modules.
func (m *Modules) Start(ctx context.Context) error {
	// start IQ and event handlers
	for _, iqHnd := range m.iqHandlers {
		if err := iqHnd.Start(ctx); err != nil {
			return err
		}
	}
	for _, evHnd := range m.eventHandlers {
		if err := evHnd.Start(ctx); err != nil {
			return err
		}
	}
	log.Infow("Started modules",
		"iq_handlers_count", len(m.iqHandlers),
		"event_handlers_count", len(m.eventHandlers),
	)
	return nil
}

// Stop stops modules.
func (m *Modules) Stop(ctx context.Context) error {
	for _, iqHnd := range m.iqHandlers {
		if err := iqHnd.Stop(ctx); err != nil {
			return err
		}
	}
	for _, evHnd := range m.eventHandlers {
		if err := evHnd.Stop(ctx); err != nil {
			return err
		}
	}
	log.Infow("Stopped modules",
		"iq_handlers_count", len(m.iqHandlers),
		"event_handlers_count", len(m.eventHandlers),
	)
	return nil
}

// IsModuleIQ returns true in case iq stanza should be handled by modules.
func (m *Modules) IsModuleIQ(iq *stravaganza.IQ) bool {
	toJID := iq.ToJID()
	replyOnBehalf := toJID.IsServer() || toJID.IsBare()
	return m.hosts.IsLocalHost(toJID.Domain()) && replyOnBehalf && (iq.IsGet() || iq.IsSet())
}

// ProcessIQ routes the iq to the corresponding iq handler module.
func (m *Modules) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	ns := iq.AllChildren()[0].Attribute(stravaganza.Namespace)
	for _, iqHnd := range m.iqHandlers {
		if !iqHnd.MatchesNamespace(ns) {
			continue
		}
		return iqHnd.ProcessIQ(ctx, iq)
	}
	// ...IQ not handled...
	resp, _ := stanzaerror.E(stanzaerror.ServiceUnavailable, iq).Stanza(false)
	_ = m.router.Route(ctx, resp)
	return nil
}

// IsEnabled tells whether a specific module it's been registered.
func (m *Modules) IsEnabled(moduleName string) bool {
	for _, iqHnd := range m.iqHandlers {
		if iqHnd.Name() == moduleName {
			return true
		}
	}
	for _, evHnd := range m.eventHandlers {
		if evHnd.Name() == moduleName {
			return true
		}
	}
	return false
}
