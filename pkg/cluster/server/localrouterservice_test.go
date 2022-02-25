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
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/ortuman/jackal/pkg/cluster/pb"
	"github.com/stretchr/testify/require"
)

func TestLocalRouterService_Route(t *testing.T) {
	// given
	lrMock := &localRouterMock{}

	var recvStanza stravaganza.Stanza
	lrMock.RouteFunc = func(stanza stravaganza.Stanza, username string, resource string) error {
		recvStanza = stanza
		return nil
	}

	srv := &localRouterService{r: lrMock}

	// when
	msg := testMessageStanza()

	_, _ = srv.Route(context.Background(), &pb.LocalRouteRequest{Stanza: msg.Proto()})

	// then
	require.Len(t, lrMock.RouteCalls(), 1)
	require.Equal(t, msg.String(), recvStanza.String())
}

func TestLocalRouterService_Disconnect(t *testing.T) {
	// given
	lrMock := &localRouterMock{}

	lrMock.DisconnectFunc = func(username string, resource string, streamErr *streamerror.Error) error {
		return nil
	}

	srv := &localRouterService{r: lrMock}

	// when
	_, _ = srv.Disconnect(context.Background(), &pb.LocalDisconnectRequest{
		Username: "ortuman",
		Resource: "yard",
		StreamError: &pb.StreamError{
			Reason: pb.StreamErrorReason_STREAM_ERROR_REASON_SYSTEM_SHUTDOWN,
		},
	})

	// then
	require.Len(t, lrMock.DisconnectCalls(), 1)
}

func testMessageStanza() *stravaganza.Message {
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()
	return msg
}
