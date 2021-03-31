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

package externalmodule

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/event"
	extmodulepb "github.com/ortuman/jackal/module/external/pb"
	"github.com/ortuman/jackal/util/stringmatcher"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestModule_Features(t *testing.T) {
	// given
	cl := &grpcClientMock{}
	cl.GetStreamFeatureFunc = func(ctx context.Context, in *extmodulepb.GetStreamFeatureRequest, opts ...grpc.CallOption) (*extmodulepb.GetStreamFeatureResponse, error) {
		f := stravaganza.NewBuilder("bind").
			WithAttribute(stravaganza.Namespace, "urn:xmpp:bidi").
			Build()
		return &extmodulepb.GetStreamFeatureResponse{
			Feature: f.Proto(),
		}, nil
	}
	cl.GetServerFeaturesFunc = func(ctx context.Context, in *extmodulepb.GetServerFeaturesRequest, opts ...grpc.CallOption) (*extmodulepb.GetServerFeaturesResponse, error) {
		return &extmodulepb.GetServerFeaturesResponse{
			Features: []string{"srv-f1"},
		}, nil
	}
	cl.GetAccountFeaturesFunc = func(ctx context.Context, in *extmodulepb.GetAccountFeaturesRequest, opts ...grpc.CallOption) (*extmodulepb.GetAccountFeaturesResponse, error) {
		return &extmodulepb.GetAccountFeaturesResponse{
			Features: []string{"acc-f1"},
		}, nil
	}
	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (extmodulepb.ModuleClient, io.Closer, error) {
		return cl, nil, nil
	}
	mod := &ExtModule{
		cl: cl,
	}

	// when
	stmFeature, _ := mod.StreamFeature(context.Background(), "jackal.im")

	srvFeatures, _ := mod.ServerFeatures(context.Background())
	accFeatures, _ := mod.AccountFeatures(context.Background())

	// then
	require.Len(t, cl.GetServerFeaturesCalls(), 1)
	require.Len(t, cl.GetAccountFeaturesCalls(), 1)

	require.Equal(t, []string{"srv-f1"}, srvFeatures)
	require.Equal(t, []string{"acc-f1"}, accFeatures)

	require.NotNil(t, stmFeature)
	require.Equal(t, "bind", stmFeature.Name())
}

func TestModule_ProcessEvent(t *testing.T) {
	// given
	var evReq *extmodulepb.ProcessEventRequest

	doneCh := make(chan struct{})
	defer close(doneCh)

	getStanzasClient := &getStanzasClientMock{}
	getStanzasClient.RecvFunc = func() (*stravaganza.PBElement, error) {
		<-doneCh
		return nil, io.EOF
	}

	cl := &grpcClientMock{}
	cl.GetStanzasFunc = func(ctx context.Context, in *extmodulepb.GetStanzasRequest, opts ...grpc.CallOption) (extmodulepb.Module_GetStanzasClient, error) {
		return getStanzasClient, nil
	}
	cl.ProcessEventFunc = func(ctx context.Context, in *extmodulepb.ProcessEventRequest, opts ...grpc.CallOption) (*extmodulepb.ProcessEventResponse, error) {
		evReq = in
		return &extmodulepb.ProcessEventResponse{}, nil
	}

	closer := &closerMock{}
	closer.CloseFunc = func() error { return nil }

	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (extmodulepb.ModuleClient, io.Closer, error) {
		return cl, closer, nil
	}

	sn := sonar.New()
	mod := &ExtModule{
		opts: Options{
			Topics: []string{event.C2SStreamIQReceived},
		},
		sonar: sn,
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
	_ = mod.Start(context.Background())

	_ = sn.Post(context.Background(), sonar.NewEventBuilder(event.C2SStreamIQReceived).
		WithInfo(&event.C2SStreamEventInfo{Stanza: iq}).
		Build(),
	)

	_ = mod.Stop(context.Background())

	// then
	require.Equal(t, event.C2SStreamIQReceived, evReq.EventName)

	require.NotNil(t, evReq.Payload)
	require.NotNil(t, evReq.GetC2SStreamEvInfo().GetStanza())
	require.Equal(t, "iq", evReq.GetC2SStreamEvInfo().GetStanza().Name)

	require.Len(t, cl.ProcessEventCalls(), 1)
	require.Len(t, closer.CloseCalls(), 1)
}

func TestModule_IQHandler(t *testing.T) {
	// given
	cl := &grpcClientMock{}
	cl.ProcessIQFunc = func(ctx context.Context, in *extmodulepb.ProcessIQRequest, opts ...grpc.CallOption) (*extmodulepb.ProcessIQResponse, error) {
		return &extmodulepb.ProcessIQResponse{}, nil
	}
	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (extmodulepb.ModuleClient, io.Closer, error) {
		return cl, nil, nil
	}
	mod := &ExtModule{
		opts: Options{
			NamespaceMatcher: stringmatcher.Any,
			IsIQHandler:      true,
		},
		cl: cl,
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
	_ = mod.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, cl.ProcessIQCalls(), 1)
}

func TestModule_MessageHandler(t *testing.T) {
	// given
	cl := &grpcClientMock{}
	cl.PreProcessMessageFunc = func(ctx context.Context, in *extmodulepb.PreProcessMessageRequest, opts ...grpc.CallOption) (*extmodulepb.PreProcessMessageResponse, error) {
		return &extmodulepb.PreProcessMessageResponse{
			Message: in.Message,
		}, nil
	}
	cl.PreRouteMessageFunc = func(ctx context.Context, in *extmodulepb.PreRouteMessageRequest, opts ...grpc.CallOption) (*extmodulepb.PreRouteMessageResponse, error) {
		return &extmodulepb.PreRouteMessageResponse{
			Message: in.Message,
		}, nil
	}
	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (extmodulepb.ModuleClient, io.Closer, error) {
		return cl, nil, nil
	}
	mod := &ExtModule{
		opts: Options{
			IsMessagePreProcessor: true,
			IsMessagePreRouter:    true,
		},
		cl: cl,
	}
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("id", "i1234")
	b.WithAttribute("from", "noelia@jabber.org/balcony")
	b.WithAttribute("to", "ortuman@jackal.im/chamber")
	b.WithAttribute("type", "chat")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg0, _ := b.BuildMessage(true)

	// when
	msg1, _ := mod.PreProcessMessage(context.Background(), msg0)
	msg2, _ := mod.PreRouteMessage(context.Background(), msg1)

	// then
	require.NotNil(t, msg1)
	require.NotNil(t, msg2)

	require.Len(t, cl.PreProcessMessageCalls(), 1)
	require.Len(t, cl.PreRouteMessageCalls(), 1)
}

func TestModule_Route(t *testing.T) {
	// given
	var mu sync.RWMutex

	recvCh := make(chan *stravaganza.PBElement)
	defer close(recvCh)

	getStanzasClient := &getStanzasClientMock{}
	getStanzasClient.RecvFunc = func() (*stravaganza.PBElement, error) {
		return <-recvCh, nil
	}

	cl := &grpcClientMock{}
	cl.GetStanzasFunc = func(ctx context.Context, in *extmodulepb.GetStanzasRequest, opts ...grpc.CallOption) (extmodulepb.Module_GetStanzasClient, error) {
		return getStanzasClient, nil
	}

	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) error {
		mu.Lock()
		defer mu.Unlock()
		respStanzas = append(respStanzas, stanza)
		return nil
	}

	closer := &closerMock{}
	closer.CloseFunc = func() error { return nil }

	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (extmodulepb.ModuleClient, io.Closer, error) {
		return cl, closer, nil
	}

	mod := &ExtModule{
		router: routerMock,
		cl:     cl,
	}

	iq1, _ := stravaganza.NewIQBuilder().
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

	iq2, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "iq0002").
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
	_ = mod.Start(context.Background())

	recvCh <- iq1.Proto()
	recvCh <- iq2.Proto()

	time.Sleep(time.Millisecond * 250) // wait until received

	_ = mod.Stop(context.Background())

	// then
	mu.Lock()
	defer mu.Unlock()

	require.Len(t, respStanzas, 2)

	require.Equal(t, "iq0001", respStanzas[0].Attribute(stravaganza.ID))
	require.Equal(t, "iq0002", respStanzas[1].Attribute(stravaganza.ID))
}
