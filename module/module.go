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
	"errors"
	"sort"

	"github.com/ortuman/jackal/event"

	"github.com/jackal-xmpp/sonar"

	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
)

// Module represents generic module interface.
type Module interface {
	// Name returns specific module name.
	Name() string

	// StreamFeature returns module stream feature element.
	StreamFeature(ctx context.Context, domain string) (stravaganza.Element, error)

	// ServerFeatures returns module server features.
	ServerFeatures(ctx context.Context) ([]string, error)

	// AccountFeatures returns module account features.
	AccountFeatures(ctx context.Context) ([]string, error)

	// Start starts module.
	Start(ctx context.Context) error

	// Stop stops module.
	Stop(ctx context.Context) error
}

// IQProcessor represents an iq processor module type.
type IQProcessor interface {
	Module

	// MatchesNamespace tells whether iq child namespace corresponds to this module.
	// The serverTarget parameter will be true in case iq target is a server entity.
	MatchesNamespace(namespace string, serverTarget bool) bool

	// ProcessIQ will be invoked whenever iq stanza should be processed by this module.
	ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error
}

// StanzaInterceptor type allows to dynamically transform stanza content.
// Interceptors may be invoked upon receiving a stanza or before sending it to the target.
type StanzaInterceptor struct {
	// ID is the interceptor identifier. Note this identifier is intended to discern which interceptor
	// is being invoked when calling InterceptStanza method, so it doesn't need to be unique across different modules.
	ID int

	// Incoming tells whether the interceptor should be invoked; either upon receiving a stanza or before sending it to the target.
	Incoming bool

	// Priority represents interceptor priority that's used to determine which interceptors should be invoked first.
	// The higher the number the more priority.
	Priority int
}

// ErrInterceptStanzaInterrupted will be returned by InterceptStanza to indicate that interception was interrupted.
var ErrInterceptStanzaInterrupted = errors.New("module: stanza interception interrupted")

// StanzaInterceptorProcessor represents an stanza interceptor module type.
type StanzaInterceptorProcessor interface {
	Module

	// Interceptors returns a set of all module interceptors.
	Interceptors() []StanzaInterceptor

	// InterceptStanza will be invoked to allow stanza transformation based on a StanzaInterceptor definition.
	// To interrupt interception ErrInterceptStanzaInterrupted should be returned.
	InterceptStanza(ctx context.Context, stanza stravaganza.Stanza, id int) (result stravaganza.Stanza, err error)
}

type stanzaInterceptor struct {
	StanzaInterceptor
	fn func(ctx context.Context, stanza stravaganza.Stanza, id int) (result stravaganza.Stanza, err error)
}

// Modules is the global module hub.
type Modules struct {
	mods             []Module
	iqProcessors     []IQProcessor
	recvInterceptors []stanzaInterceptor
	sendInterceptors []stanzaInterceptor
	hosts            hosts
	router           router.Router
	sn               *sonar.Sonar
}

// NewModules returns a new initialized Modules instance.
func NewModules(
	mods []Module,
	hosts *host.Hosts,
	router router.Router,
	sn *sonar.Sonar,
) *Modules {
	m := &Modules{
		mods:   mods,
		hosts:  hosts,
		router: router,
		sn:     sn,
	}
	m.setupModules()
	return m
}

// Start starts modules.
func (m *Modules) Start(ctx context.Context) error {
	// start modules
	var modNames []string
	for _, mod := range m.mods {
		if err := mod.Start(ctx); err != nil {
			return err
		}
		modNames = append(modNames, mod.Name())
	}
	log.Infow("Started modules",
		"iq_processors_count", len(m.iqProcessors),
		"mods_count", len(m.mods),
	)
	return m.sn.Post(ctx, sonar.NewEventBuilder(event.ModulesStarted).
		WithInfo(&event.ModulesEventInfo{
			ModuleNames: modNames,
		}).
		Build(),
	)
}

// Stop stops modules.
func (m *Modules) Stop(ctx context.Context) error {
	// stop modules
	var modNames []string
	for _, mod := range m.mods {
		if err := mod.Stop(ctx); err != nil {
			return err
		}
		modNames = append(modNames, mod.Name())
	}
	log.Infow("Stopped modules",
		"iq_processors_count", len(m.iqProcessors),
		"mods_count", len(m.mods),
	)
	return m.sn.Post(ctx, sonar.NewEventBuilder(event.ModulesStopped).
		WithInfo(&event.ModulesEventInfo{
			ModuleNames: modNames,
		}).
		Build(),
	)
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
	for _, iqHnd := range m.iqProcessors {
		if !iqHnd.MatchesNamespace(ns, iq.ToJID().IsServer()) {
			continue
		}
		return iqHnd.ProcessIQ(ctx, iq)
	}
	// ...IQ not handled...
	resp, _ := stanzaerror.E(stanzaerror.ServiceUnavailable, iq).Stanza(false)
	_, _ = m.router.Route(ctx, resp)
	return nil
}

// InterceptStanza performs module stanza transformation.
func (m *Modules) InterceptStanza(ctx context.Context, stanza stravaganza.Stanza, incoming bool) (stravaganza.Stanza, error) {
	var interceptors []stanzaInterceptor
	switch {
	case incoming:
		interceptors = m.recvInterceptors
	default:
		interceptors = m.sendInterceptors
	}
	var err error

	ts := stanza
	for _, inter := range interceptors {
		ts, err = inter.fn(ctx, ts, inter.ID)
		if err != nil {
			return nil, err
		}
	}
	return ts, nil
}

// IsEnabled tells whether a specific module it's been registered.
func (m *Modules) IsEnabled(moduleName string) bool {
	for _, mod := range m.mods {
		if mod.Name() == moduleName {
			return true
		}
	}
	return false
}

// StreamFeatures returns stream features of all registered modules.
func (m *Modules) StreamFeatures(ctx context.Context, domain string) ([]stravaganza.Element, error) {
	var sfs []stravaganza.Element
	for _, mod := range m.mods {
		sf, err := mod.StreamFeature(ctx, domain)
		if err != nil {
			return nil, err
		}
		if sf != nil {
			sfs = append(sfs, sf)
		}
	}
	return sfs, nil
}

func (m *Modules) setupModules() {
	for _, mod := range m.mods {
		iqPr, ok := mod.(IQProcessor)
		if ok {
			m.iqProcessors = append(m.iqProcessors, iqPr)
		}
		stanzaInterceptorPr, ok := mod.(StanzaInterceptorProcessor)
		if ok {
			stanzaInterceptors := stanzaInterceptorPr.Interceptors()
			for _, interceptor := range stanzaInterceptors {
				switch {
				case interceptor.Incoming:
					m.recvInterceptors = append(m.recvInterceptors, stanzaInterceptor{
						StanzaInterceptor: interceptor,
						fn:                stanzaInterceptorPr.InterceptStanza,
					})
				default:
					m.sendInterceptors = append(m.sendInterceptors, stanzaInterceptor{
						StanzaInterceptor: interceptor,
						fn:                stanzaInterceptorPr.InterceptStanza,
					})
				}
			}
		}
	}
	// sort interceptors by priority
	sort.Slice(m.recvInterceptors, func(i, j int) bool {
		return m.recvInterceptors[i].Priority > m.recvInterceptors[j].Priority
	})
	sort.Slice(m.sendInterceptors, func(i, j int) bool {
		return m.sendInterceptors[i].Priority > m.sendInterceptors[j].Priority
	})
}
