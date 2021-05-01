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

func TestQueuePush(t *testing.T) {
	// given
	q := newQueue()

	// when
	q.push(stravaganza.NewBuilder("iq").Build())
	q.push(stravaganza.NewBuilder("message").Build())

	// then
	require.Len(t, q.entries, 2)
	require.Equal(t, q.entries[0].h, uint64(1))
	require.Equal(t, q.entries[1].h, uint64(2))
}

func TestQueueAcknowledge(t *testing.T) {
	// given
	q := newQueue()

	q.push(stravaganza.NewBuilder("iq").Build())
	q.push(stravaganza.NewBuilder("presence").Build())
	q.push(stravaganza.NewBuilder("message").Build())

	// when
	q.acknowledge(2)

	// then
	require.Len(t, q.entries, 1)
	require.Equal(t, "message", q.entries[0].el.Name())
	require.Equal(t, uint64(3), q.entries[0].h)
}

func TestQueueEmpty(t *testing.T) {
	// given
	q := newQueue()

	q.push(stravaganza.NewBuilder("iq").Build())

	// when
	q.acknowledge(1)

	// then
	require.Len(t, q.entries, 0)
}

func TestQueueElements(t *testing.T) {
	// given
	q := newQueue()

	q.push(stravaganza.NewBuilder("iq").Build())
	q.push(stravaganza.NewBuilder("presence").Build())
	q.push(stravaganza.NewBuilder("message").Build())

	// when
	els := q.elements()

	// then
	require.Len(t, els, 3)
}
