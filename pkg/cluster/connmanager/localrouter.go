// Copyright 2022 The jackal Authors
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

	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	clusterpb "github.com/ortuman/jackal/pkg/cluster/pb"
)

// LocalRouter defines local router service.
type LocalRouter interface {
	Route(ctx context.Context, stanza stravaganza.Stanza, username, resource string) error
	Disconnect(ctx context.Context, username, resource string, streamErr *streamerror.Error) error
}

type localRouter struct {
	cl clusterpb.LocalRouterClient
}

func (cc *localRouter) Route(ctx context.Context, stanza stravaganza.Stanza, username, resource string) error {
	_, err := cc.cl.Route(ctx, &clusterpb.LocalRouteRequest{
		Username: username,
		Resource: resource,
		Stanza:   stanza.Proto(),
	})
	return err
}

func (cc *localRouter) Disconnect(ctx context.Context, username, resource string, streamErr *streamerror.Error) error {
	_, err := cc.cl.Disconnect(ctx,
		&clusterpb.LocalDisconnectRequest{
			Username:    username,
			Resource:    resource,
			StreamError: toProtoStreamError(streamErr),
		},
	)
	return err
}
