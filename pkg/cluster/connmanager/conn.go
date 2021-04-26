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

package clusterconnmanager

import (
	"context"
	"io"
	"strconv"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	clusterpb "github.com/ortuman/jackal/pkg/cluster/pb"
	"github.com/ortuman/jackal/pkg/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var stmErrReasonMap = map[streamerror.Reason]clusterpb.StreamErrorReason{
	streamerror.InvalidXML:             clusterpb.StreamErrorReason_STREAM_ERROR_REASON_INVALID_XML,
	streamerror.InvalidNamespace:       clusterpb.StreamErrorReason_STREAM_ERROR_REASON_INVALID_NAMESPACE,
	streamerror.HostUnknown:            clusterpb.StreamErrorReason_STREAM_ERROR_REASON_HOST_UNKNOWN,
	streamerror.Conflict:               clusterpb.StreamErrorReason_STREAM_ERROR_REASON_CONFLICT,
	streamerror.InvalidFrom:            clusterpb.StreamErrorReason_STREAM_ERROR_REASON_INVALID_FROM,
	streamerror.PolicyViolation:        clusterpb.StreamErrorReason_STREAM_ERROR_REASON_POLICY_VIOLATION,
	streamerror.RemoteConnectionFailed: clusterpb.StreamErrorReason_STREAM_ERROR_REASON_REMOTE_CONNECTION_FAILED,
	streamerror.ConnectionTimeout:      clusterpb.StreamErrorReason_STREAM_ERROR_REASON_CONNECTION_TIMEOUT,
	streamerror.UnsupportedStanzaType:  clusterpb.StreamErrorReason_STREAM_ERROR_REASON_UNSUPPORTED_STANZA_TYPE,
	streamerror.UnsupportedVersion:     clusterpb.StreamErrorReason_STREAM_ERROR_REASON_UNSUPPORTED_VERSION,
	streamerror.NotAuthorized:          clusterpb.StreamErrorReason_STREAM_ERROR_REASON_NOT_AUTHORIZED,
	streamerror.ResourceConstraint:     clusterpb.StreamErrorReason_STREAM_ERROR_REASON_RESOURCE_CONSTRAINT,
	streamerror.SystemShutdown:         clusterpb.StreamErrorReason_STREAM_ERROR_REASON_SYSTEM_SHUTDOWN,
	streamerror.UndefinedCondition:     clusterpb.StreamErrorReason_STREAM_ERROR_REASON_UNDEFINED_CONDITION,
	streamerror.InternalServerError:    clusterpb.StreamErrorReason_STREAM_ERROR_REASON_INTERNAL_SERVER_ERROR,
}

var dialFn = dialContext

type clusterConn struct {
	target string
	ver    *version.SemanticVersion
	cc     io.Closer
	lr     LocalRouter
	cr     ComponentRouter
}

func newConn(addr string, port int, ver *version.SemanticVersion) *clusterConn {
	return &clusterConn{
		target: addr + ":" + strconv.Itoa(port),
		ver:    ver,
	}
}

func (c *clusterConn) LocalRouter() LocalRouter         { return c.lr }
func (c *clusterConn) ComponentRouter() ComponentRouter { return c.cr }

func (c *clusterConn) clusterAPIVer() *version.SemanticVersion {
	return c.ver
}

func (c *clusterConn) dialContext(ctx context.Context) error {
	lr, cr, cc, err := dialFn(ctx, c.target)
	if err != nil {
		return err
	}
	c.lr = lr
	c.cr = cr
	c.cc = cc
	return nil
}

func (c *clusterConn) close() error {
	return c.cc.Close()
}

func toProtoStreamError(sErr *streamerror.Error) *clusterpb.StreamError {
	pse := &clusterpb.StreamError{
		Reason: stmErrReasonMap[sErr.Reason],
		Lang:   sErr.Lang,
		Text:   sErr.Text,
	}
	if sErr.ApplicationElement != nil {
		pse.ApplicationElement = sErr.ApplicationElement.Proto()
	}
	return pse
}

func dialContext(ctx context.Context, target string) (lr LocalRouter, cr ComponentRouter, cc io.Closer, err error) {
	grpcConn, err := grpc.DialContext(ctx,
		target,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 10,
			PermitWithoutStream: true,
		}),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	lr = &localRouter{cl: clusterpb.NewLocalRouterClient(grpcConn)}
	cr = &componentRouter{cl: clusterpb.NewComponentRouterClient(grpcConn)}
	return lr, cr, grpcConn, nil
}
