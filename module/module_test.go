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

package module

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/stretchr/testify/require"
)

func TestModules_StartStop(t *testing.T) {
	// given
	iqHndMock := &iqHandlerMock{}
	iqHndMock.StartFunc = func(ctx context.Context) error { return nil }
	iqHndMock.StopFunc = func(ctx context.Context) error { return nil }

	mods := &Modules{
		iqHandlers: []IQHandler{iqHndMock},
	}

	// when
	_ = mods.Start(context.Background())
	_ = mods.Stop(context.Background())

	// then
	require.Len(t, iqHndMock.StartCalls(), 1)
	require.Len(t, iqHndMock.StopCalls(), 1)
}

func TestModules_ProcessIQ(t *testing.T) {
	// given
	iqHndMock := &iqHandlerMock{}
	iqHndMock.MatchesNamespaceFunc = func(namespace string) bool {
		return namespace == "urn:xmpp:ping"
	}
	iqHndMock.StartFunc = func(ctx context.Context) error { return nil }
	iqHndMock.StopFunc = func(ctx context.Context) error { return nil }
	iqHndMock.ProcessIQFunc = func(ctx context.Context, iq *stravaganza.IQ) error {
		return nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(domain string) bool { return domain == "jackal.im" }

	mods := &Modules{
		iqHandlers: []IQHandler{iqHndMock},
		hosts:      hMock,
	}

	// when
	_ = mods.Start(context.Background())

	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "iq0001").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/res0001").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("ping").
				WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
				Build(),
		).
		BuildIQ(false)

	_ = mods.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, iqHndMock.MatchesNamespaceCalls(), 1)
	require.Len(t, iqHndMock.ProcessIQCalls(), 1)
}
