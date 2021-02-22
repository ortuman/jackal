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

package main

import (
	"github.com/ortuman/jackal/module"
	eventhandlerexternal "github.com/ortuman/jackal/module/eventhandler/external"
	"github.com/ortuman/jackal/module/eventhandler/offline"
	iqhandlerexternal "github.com/ortuman/jackal/module/iqhandler/external"
	"github.com/ortuman/jackal/module/iqhandler/roster"
	"github.com/ortuman/jackal/module/iqhandler/xep0030"
	"github.com/ortuman/jackal/module/iqhandler/xep0054"
	"github.com/ortuman/jackal/module/iqhandler/xep0092"
	"github.com/ortuman/jackal/module/iqhandler/xep0199"
	"github.com/ortuman/jackal/util/stringmatcher"
	stringsutil "github.com/ortuman/jackal/util/strings"
)

func initModules(a *serverApp, cfg modulesConfig) error {
	var iqHandlers []module.IQHandler
	var eventHandlers []module.EventHandler

	// roster
	if stringsutil.StringSliceContains(roster.ModuleName, cfg.Enabled) {
		ros := roster.New(a.router, a.rep, a.resMng, a.hosts, a.sonar)
		iqHandlers = append(iqHandlers, ros)
	}
	// offline
	if stringsutil.StringSliceContains(offline.ModuleName, cfg.Enabled) {
		off := offline.New(a.router, a.hosts, a.rep, a.locker, a.sonar, offline.Options{
			QueueSize: cfg.Offline.QueueSize,
		})
		eventHandlers = append(eventHandlers, off)
	}
	// version
	if stringsutil.StringSliceContains(xep0092.ModuleName, cfg.Enabled) {
		ver := xep0092.New(a.router, xep0092.Options{
			ShowOS: cfg.Version.ShowOS,
		})
		iqHandlers = append(iqHandlers, ver)
	}
	// vCard
	if stringsutil.StringSliceContains(xep0054.ModuleName, cfg.Enabled) {
		vCard := xep0054.New(a.rep, a.router, a.sonar)
		iqHandlers = append(iqHandlers, vCard)
	}
	// ping
	if stringsutil.StringSliceContains(xep0199.ModuleName, cfg.Enabled) {
		ping := xep0199.New(a.router, a.sonar, xep0199.Options{
			AckTimeout:    cfg.Ping.AckTimeout,
			Interval:      cfg.Ping.Interval,
			SendPings:     cfg.Ping.SendPings,
			TimeoutAction: cfg.Ping.TimeoutAction,
		})
		iqHandlers = append(iqHandlers, ping)
	}

	// external IQ handlers
	extIQHandlers, err := initExtIQHandlers(a, cfg.External.IQHandlers)
	if err != nil {
		return err
	}
	iqHandlers = append(iqHandlers, extIQHandlers...)

	// external event handlers
	extEventHandlers, err := initExtEventHandlers(a, cfg.External.EventHandlers)
	if err != nil {
		return err
	}
	eventHandlers = append(eventHandlers, extEventHandlers...)

	// disco
	if stringsutil.StringSliceContains("disco", cfg.Enabled) {
		var mods []module.Module
		for _, m := range iqHandlers {
			mods = append(mods, m)
		}
		for _, m := range eventHandlers {
			mods = append(mods, m)
		}
		disc := xep0030.New(a.router, mods, a.comps, a.rep, a.resMng)
		iqHandlers = append(iqHandlers, disc)
	}

	a.mods = module.NewModules(iqHandlers, eventHandlers, a.hosts, a.router)
	a.registerStartStopper(a.mods)
	return nil
}

func initExtIQHandlers(a *serverApp, configs []extIQHandlerConfig) ([]module.IQHandler, error) {
	var iqHandlers []module.IQHandler
	for _, cfg := range configs {
		nsMatcher := stringmatcher.Any
		if len(cfg.Namespace.In) > 0 {
			nsMatcher = stringmatcher.NewStringMatcher(cfg.Namespace.In)
		} else if len(cfg.Namespace.RegEx) > 0 {
			var err error
			nsMatcher, err = stringmatcher.NewRegExMatcher(cfg.Namespace.RegEx)
			if err != nil {
				return nil, err
			}
		}
		iqHandlers = append(iqHandlers, iqhandlerexternal.New(
			cfg.Address,
			cfg.IsSecure,
			nsMatcher,
			a.router,
		))
	}
	return iqHandlers, nil
}

func initExtEventHandlers(a *serverApp, configs []extEventHandlerConfig) ([]module.EventHandler, error) {
	var eventHandlers []module.EventHandler
	for _, cfg := range configs {
		eventHandlers = append(eventHandlers, eventhandlerexternal.New(
			cfg.Address,
			cfg.IsSecure,
			cfg.Topics,
			a.sonar,
		))
	}
	return eventHandlers, nil
}
