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

package eventhandlerexternal

import (
	"context"
	"sync"
	"testing"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/event"
	eventhandlerpb "github.com/ortuman/jackal/module/eventhandler/external/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestHandler_DiscoFeatures(t *testing.T) {
	// given
	cl := &grpcClientMock{}
	cl.GetDiscoFeaturesFunc = func(ctx context.Context, in *eventhandlerpb.GetDiscoFeaturesRequest, opts ...grpc.CallOption) (*eventhandlerpb.GetDiscoFeaturesResponse, error) {
		return &eventhandlerpb.GetDiscoFeaturesResponse{
			ServerFeatures:  []string{"srv-f1"},
			AccountFeatures: []string{"acc-f1"},
		}, nil
	}
	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (eventhandlerpb.EventHandlerClient, *grpc.ClientConn, error) {
		return cl, nil, nil
	}
	hnd := &Handler{}

	// when
	_ = hnd.Start(context.Background())

	// then
	require.Len(t, cl.GetDiscoFeaturesCalls(), 1)

	require.Equal(t, []string{"srv-f1"}, hnd.ServerFeatures())
	require.Equal(t, []string{"acc-f1"}, hnd.AccountFeatures())
}

func TestHandler_ProcessEvent(t *testing.T) {
	// given
	var mu sync.Mutex
	var evReq *eventhandlerpb.ProcessEventRequest

	cl := &grpcClientMock{}
	cl.GetDiscoFeaturesFunc = func(ctx context.Context, in *eventhandlerpb.GetDiscoFeaturesRequest, opts ...grpc.CallOption) (*eventhandlerpb.GetDiscoFeaturesResponse, error) {
		return &eventhandlerpb.GetDiscoFeaturesResponse{}, nil
	}
	cl.ProcessEventFunc = func(ctx context.Context, in *eventhandlerpb.ProcessEventRequest, opts ...grpc.CallOption) (*eventhandlerpb.ProcessEventResponse, error) {
		mu.Lock()
		defer mu.Unlock()
		evReq = in
		return &eventhandlerpb.ProcessEventResponse{}, nil
	}
	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (eventhandlerpb.EventHandlerClient, *grpc.ClientConn, error) {
		return cl, nil, nil
	}
	sn := sonar.New()

	hnd := &Handler{
		topics: []string{event.C2SStreamIQReceived},
		sonar:  sn,
	}

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

	// when
	_ = hnd.Start(context.Background())

	_ = sn.Post(context.Background(), sonar.NewEventBuilder(event.C2SStreamIQReceived).
		WithInfo(&event.C2SStreamEventInfo{Stanza: iq}).
		Build(),
	)

	// then
	mu.Lock()
	defer mu.Unlock()

	require.Equal(t, event.C2SStreamIQReceived, evReq.EventName)

	require.NotNil(t, evReq.Payload)

	require.NotNil(t, evReq.GetC2SStreamEvInfo().GetStanza())

	require.Equal(t, "iq", evReq.GetC2SStreamEvInfo().GetStanza().Name)

	require.Len(t, cl.GetDiscoFeaturesCalls(), 1)
	require.Len(t, cl.ProcessEventCalls(), 1)
}
