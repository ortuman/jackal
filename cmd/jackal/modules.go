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
	"fmt"

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
	"github.com/ortuman/jackal/module/xep0202"
	"github.com/ortuman/jackal/module/xep0280"
	"github.com/ortuman/jackal/util/stringmatcher"
)

var modFns = map[string]func(a *serverApp, cfg modulesConfig) module.Module{
	// Roster
	// (https://xmpp.org/rfcs/rfc6121.html#roster)
	roster.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return roster.New(a.router, a.rep, a.resMng, a.hosts, a.sonar)
	},
	// Offline
	// (https://xmpp.org/extensions/xep-0160.html)
	offline.ModuleName: func(a *serverApp, cfg modulesConfig) module.Module {
		return offline.New(a.router, a.hosts, a.rep, a.locker, a.sonar, offline.Config{
			QueueSize: cfg.Offline.QueueSize,
		})
	},
	// XEP-0012: Last Activity
	// (https://xmpp.org/extensions/xep-0012.html)
	xep0012.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0012.New(a.router, a.hosts, a.resMng, a.rep, a.sonar)
	},
	// XEP-0030: Service Discovery
	// (https://xmpp.org/extensions/xep-0030.html)
	xep0030.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0030.New(a.router, a.comps, a.rep, a.resMng, a.sonar)
	},
	// XEP-0049: Private XML Storage
	// (https://xmpp.org/extensions/xep-0049.html)
	xep0049.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0049.New(a.rep, a.router, a.sonar)
	},
	// XEP-0054: vcard-temp
	// (https://xmpp.org/extensions/xep-0054.html)
	xep0054.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0054.New(a.router, a.rep, a.sonar)
	},
	// XEP-0092: Software Version
	// (https://xmpp.org/extensions/xep-0092.html)
	xep0092.ModuleName: func(a *serverApp, cfg modulesConfig) module.Module {
		return xep0092.New(a.router, xep0092.Config{
			ShowOS: cfg.Version.ShowOS,
		})
	},
	// XEP-0115: Entity Capabilities
	// (https://xmpp.org/extensions/xep-0115.html)
	xep0115.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0115.New(a.router, a.rep, a.sonar)
	},
	// XEP-0191: Blocking Command
	// (https://xmpp.org/extensions/xep-0191.html)
	xep0191.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0191.New(a.router, a.hosts, a.resMng, a.rep, a.sonar)
	},
	// XEP-0199: XMPP Ping
	// (https://xmpp.org/extensions/xep-0199.html)
	xep0199.ModuleName: func(a *serverApp, cfg modulesConfig) module.Module {
		return xep0199.New(a.router, a.sonar, xep0199.Config{
			AckTimeout:    cfg.Ping.AckTimeout,
			Interval:      cfg.Ping.Interval,
			SendPings:     cfg.Ping.SendPings,
			TimeoutAction: cfg.Ping.TimeoutAction,
		})
	},
	// XEP-0202: Entity Time
	// (https://xmpp.org/extensions/xep-0202.html)
	xep0202.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0202.New(a.router)
	},
	// XEP-0280: Message Carbons
	// (https://xmpp.org/extensions/xep-0280.html)
	xep0280.ModuleName: func(a *serverApp, _ modulesConfig) module.Module {
		return xep0280.New(a.hosts, a.router, a.resMng, a.sonar)
	},
}

func initModules(a *serverApp, cfg modulesConfig) error {
	var mods []module.Module

	// enabled modules
	for _, mName := range cfg.Enabled {
		fn, ok := modFns[mName]
		if !ok {
			return fmt.Errorf("main: unrecognized module name: %s", mName)
		}
		mods = append(mods, fn(a, cfg))
	}
	// external modules
	for _, extCfg := range cfg.External {
		var err error
		var nsMatcher stringmatcher.Matcher

		var interceptors []module.StanzaInterceptor
		switch {
		case len(extCfg.IQHandler.Namespace.In) > 0:
			nsMatcher = stringmatcher.NewStringMatcher(extCfg.IQHandler.Namespace.In)
		case len(extCfg.IQHandler.Namespace.RegEx) > 0:
			nsMatcher, err = stringmatcher.NewRegExMatcher(extCfg.IQHandler.Namespace.RegEx)
			if err != nil {
				return err
			}
		}
		for _, interceptor := range extCfg.StanzaInterceptors {
			interceptors = append(interceptors, module.StanzaInterceptor{
				ID:       interceptor.ID,
				Incoming: interceptor.Incoming,
				Priority: interceptor.Priority,
			})
		}
		mods = append(mods, externalmodule.New(
			extCfg.Address,
			extCfg.IsSecure,
			a.router,
			a.sonar,
			externalmodule.Config{
				RequestTimeout:   extCfg.RequestTimeout,
				Topics:           extCfg.EventHandler.Topics,
				TargetEntity:     extCfg.IQHandler.TargetEntity,
				NamespaceMatcher: nsMatcher,
				Interceptors:     interceptors,
			},
		))
	}
	a.mods = module.NewModules(mods, a.hosts, a.router, a.sonar)
	a.registerStartStopper(a.mods)
	return nil
}
