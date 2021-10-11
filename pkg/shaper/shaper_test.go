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
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/util/stringmatcher"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestShaper_RateLimiter(t *testing.T) {
	// given
	s := Shaper{
		Name:        "foo",
		MaxSessions: 5,
		rateLimit:   2000,
		burst:       1000,
		jidMatcher:  stringmatcher.Any,
	}

	// when
	rLim := s.RateLimiter()

	// then
	require.Equal(t, rate.Limit(2000), rLim.Limit())
	require.Equal(t, 1000, rLim.Burst())
}

func TestShapers_MatchingJID(t *testing.T) {
	// given
	var ss Shapers
	ss = append(ss, Shaper{
		Name:        "foo",
		MaxSessions: 5,
		rateLimit:   2000,
		burst:       1000,
		jidMatcher:  stringmatcher.Any,
	})

	j, _ := jid.NewWithString("ortuman@gmail.com", true)

	// when
	s := ss.MatchingJID(j)
	rLim := s.RateLimiter()

	// then
	require.Equal(t, rate.Limit(2000), rLim.Limit())
	require.Equal(t, 1000, rLim.Burst())
}

func TestShapers_Default(t *testing.T) {
	// given
	ss := new(Shapers)

	// when
	s1 := ss.DefaultC2S()
	s2 := ss.DefaultS2S()

	// then
	require.Equal(t, &defaultC2SShaper, s1)
	require.Equal(t, &defaultS2SShaper, s2)
}
