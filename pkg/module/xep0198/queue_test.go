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

func TestManagerProcessInboundStanza(t *testing.T) {
	// given
	m, _ := newQueue(nil)

	// when
	m.processOutboundStanza(testStanza())
	m.processOutboundStanza(testStanza())

	// then
	require.Len(t, m.queue(), 2)
	require.Equal(t, m.q[0].h, uint32(1))
	require.Equal(t, m.q[1].h, uint32(2))
}

func TestManagerAcknowledge(t *testing.T) {
	// given
	m, _ := newQueue(nil)

	m.processOutboundStanza(testStanza())
	m.processOutboundStanza(testStanza())
	m.processOutboundStanza(testStanza())

	// when
	m.acknowledge(2)

	// then
	require.Len(t, m.queue(), 1)
	require.Equal(t, "iq", m.q[0].st.Name())
	require.Equal(t, uint32(3), m.q[0].h)
}

func testStanza() stravaganza.Stanza {
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "i1234").
		WithAttribute(stravaganza.Type, stravaganza.ResultType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "noelia@jackal.im/chamber").
		BuildIQ()
	return iq
}
