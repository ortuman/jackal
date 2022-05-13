// Copyright 2022 The jackal Authors
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

package dns

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestForegroundResolver(t *testing.T) {
	r := NewSRVResolver("xmpp", "tcp", "jackal.im", time.Duration(0), log.NewNopLogger())
	r.lookUpFn = func(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		return "_xmpp._tcp.jackal.im", []*net.SRV{
			{Target: "xmpp2.jackal.im.", Port: 30000},
			{Target: "xmpp0.jackal.im.", Port: 31000},
			{Target: "xmpp1.jackal.im.", Port: 32000},
		}, nil
	}

	require.NoError(t, r.Resolve(context.Background()))

	require.Equal(t, []string{"xmpp0.jackal.im:31000", "xmpp1.jackal.im:32000", "xmpp2.jackal.im:30000"}, r.Targets())
}

func TestBackgroundResolver(t *testing.T) {
	firstUpdate := []*net.SRV{
		{Target: "xmpp2.jackal.im.", Port: 30000},
		{Target: "xmpp0.jackal.im.", Port: 31000},
		{Target: "xmpp1.jackal.im.", Port: 32000},
	}
	secondUpdate := []*net.SRV{
		{Target: "xmpp3.jackal.im.", Port: 34000},
		{Target: "xmpp0.jackal.im.", Port: 31000},
	}
	var i int
	lookUpFn := func(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error) {
		if i == 0 {
			i++
			return "_xmpp._tcp.jackal.im", firstUpdate, nil
		}
		return "_xmpp._tcp.jackal.im", secondUpdate, nil
	}

	r := NewSRVResolver("xmpp", "tcp", "jackal.im", time.Second, log.NewNopLogger())
	r.lookUpFn = lookUpFn

	require.NoError(t, r.Resolve(context.Background()))
	require.Equal(t, []string{"xmpp0.jackal.im:31000", "xmpp1.jackal.im:32000", "xmpp2.jackal.im:30000"}, r.Targets())

	update := <-r.Update()

	require.ElementsMatch(t, []string{"xmpp3.jackal.im:34000"}, update.NewTargets)
	require.ElementsMatch(t, []string{"xmpp1.jackal.im:32000", "xmpp2.jackal.im:30000"}, update.OldTargets)

	require.Equal(t, []string{"xmpp0.jackal.im:31000", "xmpp3.jackal.im:34000"}, r.Targets())
}

func TestParseRecord(t *testing.T) {
	tcs := map[string]struct {
		rec          string
		service      string
		proto        string
		target       string
		expectsError error
	}{
		"well formed SRV record": {
			rec:          "_xmpp._tcp.jackal.im",
			service:      "xmpp",
			proto:        "tcp",
			target:       "jackal.im",
			expectsError: nil,
		},
		"missing service": {
			rec:          "_tcp.jackal.im",
			expectsError: errBadSRVFormat,
		},
		"missing service & proto": {
			rec:          "jackal.im",
			expectsError: errBadSRVFormat,
		},
	}
	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			srv, proto, target, err := ParseSRVRecord(tc.rec)

			require.ErrorIs(t, tc.expectsError, err)

			require.Equal(t, tc.service, srv)
			require.Equal(t, tc.proto, proto)
			require.Equal(t, tc.target, target)
		})
	}
}
