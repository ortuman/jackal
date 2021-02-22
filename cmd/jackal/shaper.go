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

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/shaper"
	"github.com/ortuman/jackal/util/stringmatcher"
)

func initShapers(a *serverApp, configs []shaperConfig) error {
	a.shapers = make(shaper.Shapers, 0)
	for _, cfg := range configs {
		var jidMatcher = stringmatcher.Any
		if len(cfg.Matching.JID.In) > 0 {
			jidMatcher = stringmatcher.NewStringMatcher(cfg.Matching.JID.In)
		} else if len(cfg.Matching.JID.RegEx) > 0 {
			var err error
			jidMatcher, err = stringmatcher.NewRegExMatcher(cfg.Matching.JID.RegEx)
			if err != nil {
				return err
			}
		}
		a.shapers = append(a.shapers, shaper.New(cfg.MaxSessions, cfg.Rate.Limit, cfg.Rate.Burst, jidMatcher))

		log.Infow(fmt.Sprintf("Registered '%s' shaper configuration", cfg.Name),
			"name", cfg.Name,
			"max_sessions", cfg.MaxSessions,
			"limit", cfg.Rate.Limit,
			"burst", cfg.Rate.Burst)
	}
	return nil
}
