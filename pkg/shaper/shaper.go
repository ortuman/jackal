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

package shaper

import (
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/util/stringmatcher"
	"golang.org/x/time/rate"
)

var defaultC2SShaper = Shaper{
	MaxSessions: 3,
	rateLimit:   131072,
	burst:       65536,
}

var defaultS2SShaper = Shaper{
	rateLimit: 262144,
	burst:     131072,
}

// Shapers represents a shaper collection ordered by priority.
type Shapers []Shaper

// MatchingJID returns the shaper that should be applied to a given JID.
func (ss Shapers) MatchingJID(j *jid.JID) *Shaper {
	for _, s := range ss {
		if s.jidMatcher.Matches(j.String()) {
			return &s
		}
	}
	if j != nil && j.IsServer() {
		return &defaultS2SShaper
	}
	return &defaultC2SShaper
}

// DefaultC2S returns C2S default shaper.
func (ss Shapers) DefaultC2S() *Shaper {
	return &defaultC2SShaper
}

// DefaultS2S returns S2S default shaper.
func (ss Shapers) DefaultS2S() *Shaper {
	return &defaultS2SShaper
}

// Shaper represents a connection traffic constraint set.
type Shaper struct {
	// Name is the shaper name.
	Name string

	// MaxSessions represents maximum sessions count.
	MaxSessions int

	rateLimit, burst int
	jidMatcher       stringmatcher.Matcher
}

// Config contains Shaper configuration parameters.
type Config struct {
	Name        string `fig:"name"`
	MaxSessions int    `fig:"max_sessions" default:"10"`
	Rate        struct {
		Limit int `fig:"limit" default:"1000"`
		Burst int `fig:"burst" default:"0"`
	} `fig:"rate"`
	Matching struct {
		JID struct {
			In    []string `fig:"in"`
			RegEx string   `fig:"regex"`
		}
	} `fig:"matching"`
}

// New returns a new Shaper given a configuration.
func New(cfg Config) (Shaper, error) {
	var jidMatcher stringmatcher.Matcher
	switch {
	case len(cfg.Matching.JID.In) > 0:
		jidMatcher = stringmatcher.NewStringMatcher(cfg.Matching.JID.In)
	case len(cfg.Matching.JID.RegEx) > 0:
		var err error
		jidMatcher, err = stringmatcher.NewRegExMatcher(cfg.Matching.JID.RegEx)
		if err != nil {
			return Shaper{}, err
		}
	default:
		jidMatcher = stringmatcher.Any
	}
	return Shaper{
		Name:        cfg.Name,
		MaxSessions: cfg.MaxSessions,
		rateLimit:   cfg.Rate.Limit,
		burst:       cfg.Rate.Burst,
		jidMatcher:  jidMatcher,
	}, nil
}

// RateLimiter returns a new rate limiter configured with shaper parameters.
func (s *Shaper) RateLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(s.rateLimit), s.burst)
}
