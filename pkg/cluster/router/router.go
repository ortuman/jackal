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

package clusterrouter

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"
)

// Router defines cluster router type.
type Router struct {
	connMng clusterConnManager
}

// New returns a new initialized Router instance.
func New(connMng *clusterconnmanager.Manager) *Router {
	r := &Router{
		connMng: connMng,
	}
	return r
}

// Route routes an XMPP stanza to a cluster remote resource.
func (r *Router) Route(ctx context.Context, stanza stravaganza.Stanza, username, resource, instanceID string) error {
	conn, err := r.connMng.GetConnection(instanceID)
	if err != nil {
		return err
	}
	return conn.LocalRouter().Route(ctx, stanza, username, resource)
}

// Disconnect performs remote cluster resource disconnection.
func (r *Router) Disconnect(ctx context.Context, username, resource string, streamErr *streamerror.Error, instanceID string) error {
	conn, err := r.connMng.GetConnection(instanceID)
	if err != nil {
		return err
	}
	return conn.LocalRouter().Disconnect(ctx, username, resource, streamErr)
}

// Start starts cluster router.
func (r *Router) Start(_ context.Context) error { return nil }

// Stop stops cluster router.
func (r *Router) Stop(_ context.Context) error { return nil }
