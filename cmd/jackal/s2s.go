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

import "github.com/ortuman/jackal/s2s"

func initS2S(a *serverApp, cfg s2sOutConfig) {
	a.s2sOutProvider = s2s.NewOutProvider(a.hosts, a.kv, a.shapers, a.sonar, s2s.Options{
		DialTimeout:    cfg.DialTimeout,
		DialbackSecret: cfg.DialbackSecret,
		ConnectTimeout: cfg.ConnectTimeout,
		KeepAlive:      cfg.KeepAlive,
		RequestTimeout: cfg.RequestTimeout,
		MaxStanzaSize:  cfg.MaxStanzaSize,
	})
	a.s2sInHub = s2s.NewInHub()

	a.registerStartStopper(a.s2sOutProvider)
	a.registerStartStopper(a.s2sInHub)
}
