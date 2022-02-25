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

package clusterserver

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/pkg/cluster/pb"
	"github.com/stretchr/testify/require"
)

func TestComponentRouterService_Route(t *testing.T) {
	// given
	csMock := &componentsMock{}

	var recvStanza stravaganza.Stanza
	csMock.ProcessStanzaFunc = func(ctx context.Context, stanza stravaganza.Stanza) error {
		recvStanza = stanza
		return nil
	}

	srv := &componentRouterService{comps: csMock}

	// when
	msg := testMessageStanza()

	_, _ = srv.Route(context.Background(), &pb.ComponentRouteRequest{Stanza: msg.Proto()})

	// then
	require.Len(t, csMock.ProcessStanzaCalls(), 1)
	require.Equal(t, msg.String(), recvStanza.String())
}
