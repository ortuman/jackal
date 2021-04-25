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
	"crypto/tls"
	"fmt"

	"github.com/ortuman/jackal/auth"
	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/component/xep0114"
	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/transport/compress"
)

var cmpLevelMap = map[string]compress.Level{
	"default": compress.DefaultCompression,
	"best":    compress.BestCompression,
	"speed":   compress.SpeedCompression,
}

var resConflictMap = map[string]c2s.ResourceConflict{
	"override":      c2s.Override,
	"disallow":      c2s.Disallow,
	"terminate_old": c2s.TerminateOld,
}

var lnFns = map[string]func(a *serverApp, cfg listenerConfig) startStopper{
	c2sListenerType: func(a *serverApp, cfg listenerConfig) startStopper {
		var extAuth *auth.External
		if len(cfg.SASL.External.Address) > 0 {
			extAuth = auth.NewExternal(
				cfg.SASL.External.Address,
				cfg.SASL.External.IsSecure,
			)
		}
		return c2s.NewSocketListener(
			cfg.BindAddr,
			cfg.Port,
			cfg.SASL.Mechanisms,
			extAuth,
			a.hosts,
			a.router,
			a.comps,
			a.mods,
			a.resMng,
			a.rep,
			a.peppers,
			a.shapers,
			a.sonar,
			c2s.Config{
				ConnectTimeout:   cfg.ConnectTimeout,
				KeepAliveTimeout: cfg.KeepAliveTimeout,
				RequestTimeout:   cfg.RequestTimeout,
				MaxStanzaSize:    cfg.MaxStanzaSize,
				CompressionLevel: cmpLevelMap[cfg.CompressionLevel],
				ResourceConflict: resConflictMap[cfg.ResourceConflict],
				UseTLS:           cfg.DirectTLS,
				TLSConfig: &tls.Config{
					Certificates: a.hosts.Certificates(),
					MinVersion:   tls.VersionTLS12,
				},
			},
		)
	},
	s2sListenerType: func(a *serverApp, cfg listenerConfig) startStopper {
		return s2s.NewSocketListener(
			cfg.BindAddr,
			cfg.Port,
			a.hosts,
			a.router,
			a.comps,
			a.mods,
			a.s2sOutProvider,
			a.s2sInHub,
			a.kv,
			a.shapers,
			a.sonar,
			s2s.Config{
				ConnectTimeout: cfg.ConnectTimeout,
				KeepAlive:      cfg.KeepAliveTimeout,
				RequestTimeout: cfg.RequestTimeout,
				MaxStanzaSize:  cfg.MaxStanzaSize,
				UseTLS:         cfg.DirectTLS,
				TLSConfig: &tls.Config{
					Certificates: a.hosts.Certificates(),
					ClientAuth:   tls.RequireAndVerifyClientCert,
					MinVersion:   tls.VersionTLS12,
				},
			},
		)
	},
	componentListenerType: func(a *serverApp, cfg listenerConfig) startStopper {
		return xep0114.NewSocketListener(
			cfg.BindAddr,
			cfg.Port,
			a.hosts,
			a.comps,
			a.extCompMng,
			a.router,
			a.shapers,
			a.sonar,
			xep0114.Config{
				ConnectTimeout:   cfg.ConnectTimeout,
				KeepAliveTimeout: cfg.KeepAliveTimeout,
				RequestTimeout:   cfg.RequestTimeout,
				MaxStanzaSize:    cfg.MaxStanzaSize,
				Secret:           cfg.Secret,
			},
		)
	},
}

func initListeners(a *serverApp, configs []listenerConfig) error {
	for _, cfg := range configs {
		lnFn, ok := lnFns[cfg.Type]
		if !ok {
			return fmt.Errorf("main: unrecognized listener: %s", cfg.Type)
		}
		ln := lnFn(a, cfg)
		a.registerStartStopper(ln)
	}
	return nil
}
