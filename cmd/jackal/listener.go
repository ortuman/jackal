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

	"github.com/ortuman/jackal/pkg/component/xep0114"
	"github.com/ortuman/jackal/pkg/s2s"
	"github.com/ortuman/jackal/pkg/transport/compress"
)

const (
	s2sListener       = "s2s"
	componentListener = "component"
)

var cmpLevelMap = map[string]compress.Level{
	"default": compress.DefaultCompression,
	"best":    compress.BestCompression,
	"speed":   compress.SpeedCompression,
}

var lnFns = map[string]func(a *serverApp, cfg listenerConfig) startStopper{
	s2sListener: func(a *serverApp, cfg listenerConfig) startStopper {
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
			a.hk,
			s2s.Config{
				ConnectTimeout:   cfg.ConnectTimeout,
				KeepAliveTimeout: cfg.KeepAliveTimeout,
				RequestTimeout:   cfg.RequestTimeout,
				MaxStanzaSize:    cfg.MaxStanzaSize,
				DirectTLS:        cfg.DirectTLS,
				TLSConfig: &tls.Config{
					Certificates: a.hosts.Certificates(),
					ClientAuth:   tls.RequireAndVerifyClientCert,
					MinVersion:   tls.VersionTLS12,
				},
			},
		)
	},
	componentListener: func(a *serverApp, cfg listenerConfig) startStopper {
		return xep0114.NewSocketListener(
			cfg.BindAddr,
			cfg.Port,
			a.hosts,
			a.comps,
			a.extCompMng,
			a.router,
			a.shapers,
			a.hk,
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
