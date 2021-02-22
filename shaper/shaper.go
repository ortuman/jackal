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
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/util/stringmatcher"
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
	// MaxSessions represens maximum sessions count.
	MaxSessions int

	rateLimit, burst int
	jidMatcher       stringmatcher.Matcher
}

// New returns a new Shaper given a configuration.
func New(maxSessions int, rateLimit int, burst int, jidMatcher stringmatcher.Matcher) Shaper {
	return Shaper{
		MaxSessions: maxSessions,
		rateLimit:   rateLimit,
		burst:       burst,
		jidMatcher:  jidMatcher,
	}
}

// RateLimiter returns a new rate limiter configured with shaper parameters.
func (s *Shaper) RateLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(s.rateLimit), s.burst)
}
