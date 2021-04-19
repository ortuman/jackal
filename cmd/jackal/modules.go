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
	externalmodule "github.com/ortuman/jackal/module/external"
	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/module/xep0012"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/module/xep0049"
	"github.com/ortuman/jackal/module/xep0054"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0115"
	"github.com/ortuman/jackal/module/xep0191"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/module/xep0280"
	"github.com/ortuman/jackal/util/stringmatcher"
	stringsutil "github.com/ortuman/jackal/util/strings"
)

func initModules(a *serverApp, cfg modulesConfig) error {
	var mods []module.Module

	// disco
	var disc *xep0030.Disco
	if stringsutil.StringSliceContains(xep0030.ModuleName, cfg.Enabled) {
		disc = xep0030.New(a.router, a.comps, a.rep, a.resMng)
		mods = append(mods, disc)
	}

	// roster
	if stringsutil.StringSliceContains(roster.ModuleName, cfg.Enabled) {
		ros := roster.New(a.router, a.rep, a.resMng, a.hosts, a.sonar)
		mods = append(mods, ros)
	}
	// offline
	if stringsutil.StringSliceContains(offline.ModuleName, cfg.Enabled) {
		off := offline.New(a.router, a.hosts, a.rep, a.locker, a.sonar, offline.Options{
			QueueSize: cfg.Offline.QueueSize,
		})
		mods = append(mods, off)
	}
	// last
	if stringsutil.StringSliceContains(xep0012.ModuleName, cfg.Enabled) {
		last := xep0012.New(a.router, a.resMng, a.rep, a.sonar)
		mods = append(mods, last)
	}
	// version
	if stringsutil.StringSliceContains(xep0092.ModuleName, cfg.Enabled) {
		ver := xep0092.New(a.router, xep0092.Options{
			ShowOS: cfg.Version.ShowOS,
		})
		mods = append(mods, ver)
	}
	// private
	if stringsutil.StringSliceContains(xep0049.ModuleName, cfg.Enabled) {
		private := xep0049.New(a.rep, a.router, a.sonar)
		mods = append(mods, private)
	}
	// vCard
	if stringsutil.StringSliceContains(xep0054.ModuleName, cfg.Enabled) {
		vCard := xep0054.New(a.router, a.rep, a.sonar)
		mods = append(mods, vCard)
	}
	// capabilities
	if stringsutil.StringSliceContains(xep0115.ModuleName, cfg.Enabled) {
		caps := xep0115.New(disc, a.router, a.rep, a.sonar)
		mods = append(mods, caps)
	}
	// blocklist
	if stringsutil.StringSliceContains(xep0191.ModuleName, cfg.Enabled) {
		blockList := xep0191.New(a.router, a.hosts, a.resMng, a.rep, a.sonar)
		mods = append(mods, blockList)
	}
	// ping
	if stringsutil.StringSliceContains(xep0199.ModuleName, cfg.Enabled) {
		ping := xep0199.New(a.router, a.sonar, xep0199.Options{
			AckTimeout:    cfg.Ping.AckTimeout,
			Interval:      cfg.Ping.Interval,
			SendPings:     cfg.Ping.SendPings,
			TimeoutAction: cfg.Ping.TimeoutAction,
		})
		mods = append(mods, ping)
	}
	// carbons
	if stringsutil.StringSliceContains(xep0280.ModuleName, cfg.Enabled) {
		carbons := xep0280.New(a.hosts, a.router, a.resMng, a.sonar)
		mods = append(mods, carbons)
	}
	// external modules
	extModules, err := initExtModules(a, cfg.External)
	if err != nil {
		return err
	}
	mods = append(mods, extModules...)

	// set disco info modules
	if disc != nil {
		disc.SetModules(mods)
	}
	a.mods = module.NewModules(mods, a.hosts, a.router)
	a.registerStartStopper(a.mods)
	return nil
}

func initExtModules(a *serverApp, configs []extModuleConfig) ([]module.Module, error) {
	var extMods []module.Module

	for _, cfg := range configs {
		var opts externalmodule.Options

		opts.RequestTimeout = cfg.RequestTimeout
		opts.Topics = cfg.EventHandler.Topics
		switch {
		case len(cfg.IQHandler.Namespace.In) > 0:
			opts.NamespaceMatcher = stringmatcher.NewStringMatcher(cfg.IQHandler.Namespace.In)
		case len(cfg.IQHandler.Namespace.RegEx) > 0:
			nsMatcher, err := stringmatcher.NewRegExMatcher(cfg.IQHandler.Namespace.RegEx)
			if err != nil {
				return nil, err
			}
			opts.NamespaceMatcher = nsMatcher
		}
		opts.TargetEntity = cfg.IQHandler.TargetEntity

		for _, interceptor := range cfg.StanzaInterceptors {
			opts.Interceptors = append(opts.Interceptors, module.StanzaInterceptor{
				ID:       interceptor.ID,
				Incoming: interceptor.Incoming,
				Priority: interceptor.Priority,
			})
		}

		extMods = append(extMods, externalmodule.New(cfg.Address, cfg.IsSecure, a.router, a.sonar, opts))
	}
	return extMods, nil
}
