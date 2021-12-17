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

	"github.com/jackal-xmpp/stravaganza/v2"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/c2s_new"
	"github.com/ortuman/jackal/pkg/cluster/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errReasonMap = map[pb.StreamErrorReason]streamerror.Reason{
	pb.StreamErrorReason_STREAM_ERROR_REASON_INVALID_XML:              streamerror.InvalidXML,
	pb.StreamErrorReason_STREAM_ERROR_REASON_INVALID_NAMESPACE:        streamerror.InvalidNamespace,
	pb.StreamErrorReason_STREAM_ERROR_REASON_HOST_UNKNOWN:             streamerror.HostUnknown,
	pb.StreamErrorReason_STREAM_ERROR_REASON_INVALID_FROM:             streamerror.InvalidFrom,
	pb.StreamErrorReason_STREAM_ERROR_REASON_CONFLICT:                 streamerror.Conflict,
	pb.StreamErrorReason_STREAM_ERROR_REASON_POLICY_VIOLATION:         streamerror.PolicyViolation,
	pb.StreamErrorReason_STREAM_ERROR_REASON_REMOTE_CONNECTION_FAILED: streamerror.RemoteConnectionFailed,
	pb.StreamErrorReason_STREAM_ERROR_REASON_CONNECTION_TIMEOUT:       streamerror.ConnectionTimeout,
	pb.StreamErrorReason_STREAM_ERROR_REASON_UNSUPPORTED_STANZA_TYPE:  streamerror.UnsupportedStanzaType,
	pb.StreamErrorReason_STREAM_ERROR_REASON_UNSUPPORTED_VERSION:      streamerror.UnsupportedVersion,
	pb.StreamErrorReason_STREAM_ERROR_REASON_NOT_AUTHORIZED:           streamerror.NotAuthorized,
	pb.StreamErrorReason_STREAM_ERROR_REASON_RESOURCE_CONSTRAINT:      streamerror.ResourceConstraint,
	pb.StreamErrorReason_STREAM_ERROR_REASON_SYSTEM_SHUTDOWN:          streamerror.SystemShutdown,
	pb.StreamErrorReason_STREAM_ERROR_REASON_UNDEFINED_CONDITION:      streamerror.UndefinedCondition,
	pb.StreamErrorReason_STREAM_ERROR_REASON_INTERNAL_SERVER_ERROR:    streamerror.InternalServerError,
}

type localRouterService struct {
	r localRouter
}

func newLocalRouterService(r *c2s_new.LocalRouter) *localRouterService {
	return &localRouterService{r: r}
}

func (s *localRouterService) Route(_ context.Context, req *pb.LocalRouteRequest) (*pb.LocalRouteResponse, error) {
	st, err := stravaganza.NewBuilderFromProto(req.GetStanza()).
		BuildStanza()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	_ = s.r.Route(st, req.GetUsername(), req.GetResource())
	return &pb.LocalRouteResponse{}, nil
}

func (s *localRouterService) Disconnect(_ context.Context, req *pb.LocalDisconnectRequest) (*pb.LocalDisconnectResponse, error) {
	_ = s.r.Disconnect(req.GetUsername(), req.GetResource(), toStreamError(req.GetStreamError()))
	return &pb.LocalDisconnectResponse{}, nil
}

func toStreamError(pbStreamErr *pb.StreamError) *streamerror.Error {
	se := &streamerror.Error{
		Reason: errReasonMap[pbStreamErr.Reason],
		Lang:   pbStreamErr.GetLang(),
		Text:   pbStreamErr.GetText(),
	}
	pbAppElement := pbStreamErr.GetApplicationElement()
	if pbAppElement != nil {
		se.ApplicationElement = stravaganza.NewBuilderFromProto(pbAppElement).Build()
	}
	return se
}
