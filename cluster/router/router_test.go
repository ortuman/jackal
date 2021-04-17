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

package clusterrouter

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	clusterconnmanager "github.com/ortuman/jackal/cluster/connmanager"
	"github.com/stretchr/testify/require"
)

func TestRouter_Route(t *testing.T) {
	// given
	lrMock := &localRouterMock{}
	lrMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza, username string, resource string) error {
		return nil
	}

	connMock := &clusterConnMock{}
	connMock.LocalRouterFunc = func() clusterconnmanager.LocalRouter {
		return lrMock
	}
	connMngMock := &clusterConnManagerMock{}
	connMngMock.GetConnectionFunc = func(instanceID string) (clusterconnmanager.Conn, error) {
		if instanceID == "a1234" {
			return connMock, nil
		}
		return nil, clusterconnmanager.ErrConnNotFound
	}

	r := &Router{
		connMng: connMngMock,
	}
	// when
	_ = r.Route(context.Background(), testMessageStanza(), "ortuman", "balcony", "a1234")

	// then
	require.Len(t, lrMock.RouteCalls(), 1)
}

func TestRouter_Disconnect(t *testing.T) {
	// given
	lrMock := &localRouterMock{}
	lrMock.DisconnectFunc = func(ctx context.Context, username string, resource string, streamErr *streamerror.Error) error {
		return nil
	}

	connMock := &clusterConnMock{}
	connMock.LocalRouterFunc = func() clusterconnmanager.LocalRouter {
		return lrMock
	}
	connMngMock := &clusterConnManagerMock{}
	connMngMock.GetConnectionFunc = func(instanceID string) (clusterconnmanager.Conn, error) {
		if instanceID == "a1234" {
			return connMock, nil
		}
		return nil, clusterconnmanager.ErrConnNotFound
	}

	r := &Router{
		connMng: connMngMock,
	}
	// when
	_ = r.Disconnect(context.Background(), "u1", "r1", streamerror.E(streamerror.SystemShutdown), "a1234")

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
