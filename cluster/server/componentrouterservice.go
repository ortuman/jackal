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
	"errors"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/cluster/pb"
	"github.com/ortuman/jackal/component"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type componentRouterService struct {
	comps components
}

func newComponentRouterService(comps *component.Components) *componentRouterService {
	return &componentRouterService{
		comps: comps,
	}
}

func (s *componentRouterService) Route(ctx context.Context, req *pb.ComponentRouteRequest) (*pb.ComponentRouteResponse, error) {
	st, err := stravaganza.NewBuilderFromProto(req.GetStanza()).
		BuildStanza(true)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.comps.ProcessStanza(ctx, st); err != nil {
		switch {
		case errors.Is(err, component.ErrComponentNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &pb.ComponentRouteResponse{}, nil
}
