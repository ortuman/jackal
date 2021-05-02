// Copyright 2021 The jackal Authors
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

package xep0198

import (
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/stretchr/testify/require"
)

func TestManagerQueuePush(t *testing.T) {
	// given
	m := newManager()

	// when
	m.pushElement(stravaganza.NewBuilder("iq").Build())
	m.pushElement(stravaganza.NewBuilder("message").Build())

	// then
	require.Len(t, m.q.entries, 2)
	require.Equal(t, m.q.entries[0].outH, uint32(1))
	require.Equal(t, m.q.entries[1].outH, uint32(2))
}

func TestManagerQueueAcknowledge(t *testing.T) {
	// given
	m := newManager()

	m.pushElement(stravaganza.NewBuilder("iq").Build())
	m.pushElement(stravaganza.NewBuilder("presence").Build())
	m.pushElement(stravaganza.NewBuilder("message").Build())

	// when
	m.acknowledge(2)

	// then
	require.Len(t, m.q.entries, 1)
	require.Equal(t, "message", m.q.entries[0].el.Name())
	require.Equal(t, uint32(3), m.q.entries[0].outH)
}

func TestManagerQueueEmpty(t *testing.T) {
	// given
	m := newManager()

	m.pushElement(stravaganza.NewBuilder("iq").Build())

	// when
	m.acknowledge(1)

	// then
	require.Len(t, m.q.entries, 0)
}
