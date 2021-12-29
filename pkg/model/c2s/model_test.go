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

package c2smodel

import (
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/stretchr/testify/require"
)

func TestInfo_Get(t *testing.T) {
	// given
	m := map[string]string{
		"k1": "v1",
		"k2": "true",
		"k3": "46",
		"k4": "2.24532",
	}

	// when
	i := Info{M: m}

	// then
	require.Equal(t, "v1", i.String("k1"))
	require.Equal(t, true, i.Bool("k2"))
	require.Equal(t, 46, i.Int("k3"))
	require.Equal(t, 2.24532, i.Float("k4"))
}

func TestResource_Presence(t *testing.T) {
	// given
	jd, _ := jid.NewWithString("ortuman@jackal.im", true)

	pr := xmpputil.MakePresence(jd, jd, stravaganza.AvailableType, []stravaganza.Element{
		stravaganza.NewBuilder("priority").
			WithText("10").
			Build(),
	})

	r := NewResourceDesc("i0", nil, pr, Info{})

	// when
	avail := r.IsAvailable()
	prio := r.Priority()

	// then
	require.True(t, avail)
	require.Equal(t, prio, int8(10))
}
