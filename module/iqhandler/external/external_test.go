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

package iqhandlerexternal

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	iqhandlerpb "github.com/ortuman/jackal/module/iqhandler/external/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestHandler_DiscoFeatures(t *testing.T) {
	// given
	cl := &grpcClientMock{}
	cl.GetDiscoFeaturesFunc = func(ctx context.Context, in *iqhandlerpb.GetDiscoFeaturesRequest, opts ...grpc.CallOption) (*iqhandlerpb.GetDiscoFeaturesResponse, error) {
		return &iqhandlerpb.GetDiscoFeaturesResponse{
			ServerFeatures:  []string{"srv-f1"},
			AccountFeatures: []string{"acc-f1"},
		}, nil
	}
	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (iqhandlerpb.IQHandlerClient, *grpc.ClientConn, error) {
		return cl, nil, nil
	}
	hnd := &Handler{}

	// when
	_ = hnd.Start(context.Background())

	// then
	require.Equal(t, []string{"srv-f1"}, hnd.ServerFeatures())
	require.Equal(t, []string{"acc-f1"}, hnd.AccountFeatures())
}

func TestHandler_ProcessIQ(t *testing.T) {
	// given
	cl := &grpcClientMock{}
	cl.GetDiscoFeaturesFunc = func(ctx context.Context, in *iqhandlerpb.GetDiscoFeaturesRequest, opts ...grpc.CallOption) (*iqhandlerpb.GetDiscoFeaturesResponse, error) {
		return &iqhandlerpb.GetDiscoFeaturesResponse{}, nil
	}
	cl.ProcessIQFunc = func(ctx context.Context, in *iqhandlerpb.ProcessIQRequest, opts ...grpc.CallOption) (*iqhandlerpb.ProcessIQResponse, error) {
		return &iqhandlerpb.ProcessIQResponse{}, nil
	}
	dialExtConnFn = func(ctx context.Context, addr string, isSecure bool) (iqhandlerpb.IQHandlerClient, *grpc.ClientConn, error) {
		return cl, nil, nil
	}
	hnd := &Handler{}

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
	_ = hnd.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, cl.ProcessIQCalls(), 1)
}
