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

package component

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/stretchr/testify/require"
)

func TestComponents_Components(t *testing.T) {
	// given
	compMock := &componentMock{}
	compMock.HostFunc = func() string {
		return "muc.jackal.im"
	}
	compMock.StartFunc = func(_ context.Context) error { return nil }

	cs := NewComponents(nil)

	// when
	_ = cs.Start(context.Background())
	_ = cs.RegisterComponent(context.Background(), compMock)

	// then
	require.NotNil(t, cs.Component("muc.jackal.im"))
	require.Len(t, cs.AllComponents(), 1)
}

func TestComponents_RegisterComponent(t *testing.T) {
	// given
	compMock := &componentMock{}
	compMock.HostFunc = func() string {
		return "muc.jackal.im"
	}
	compMock.StartFunc = func(_ context.Context) error { return nil }
	compMock.StopFunc = func(_ context.Context) error { return nil }

	cs := NewComponents(nil)

	// when
	_ = cs.Start(context.Background())

	_ = cs.RegisterComponent(context.Background(), compMock)
	ok1 := cs.IsComponentHost("muc.jackal.im")

	_ = cs.UnregisterComponent(context.Background(), "muc.jackal.im")
	ok2 := cs.IsComponentHost("muc.jackal.im")

	// then
	require.True(t, ok1)
	require.False(t, ok2)

	require.Len(t, compMock.StartCalls(), 1)
	require.Len(t, compMock.StopCalls(), 1)
}

func TestComponents_ProcessStanza(t *testing.T) {
	// given
	compMock := &componentMock{}
	compMock.HostFunc = func() string {
		return "muc.jackal.im"
	}
	compMock.StartFunc = func(_ context.Context) error { return nil }
	compMock.ProcessStanzaFunc = func(ctx context.Context, stanza stravaganza.Stanza) error { return nil }

	cs := NewComponents(nil)

	// when
	_ = cs.Start(context.Background())
	_ = cs.RegisterComponent(context.Background(), compMock)

	msg := testMessageStanza()
	_ = cs.ProcessStanza(context.Background(), msg)

	// then
	require.Len(t, compMock.ProcessStanzaCalls(), 1)
}

func testMessageStanza() *stravaganza.Message {
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "coven@muc.jackal.im/firstwitch")
	b.WithAttribute("to", "hag66@muc.jackal.im/secondwitch")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage(true)
	return msg
}
