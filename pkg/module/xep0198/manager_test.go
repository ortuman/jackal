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

/*
func TestStanzaQueuePush(t *testing.T) {
	// given
	m := new

	// when
	q.pro(testStanza())
	q.push(testStanza())

	// then
	require.Equal(t, 2, q.len())
	require.Equal(t, q.queue[0].h, uint32(1))
	require.Equal(t, q.queue[1].h, uint32(2))
}

func TestStanzaQueueAcknowledge(t *testing.T) {
	// given
	q := &stanzaQueue{}

	q.push(testStanza())
	q.push(testStanza())
	q.push(testStanza())

	// when
	q.acknowledge(2)

	// then
	require.Equal(t, 1, q.len())
	require.Equal(t, "iq", q.queue[0].st.Name())
	require.Equal(t, uint32(3), q.queue[0].h)
}

func TestStanzaQueueEmpty(t *testing.T) {
	// given
	q := &stanzaQueue{}

	q.push(testStanza())

	// when
	q.acknowledge(1)

	// then
	require.Equal(t, 0, q.len())
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
*/
