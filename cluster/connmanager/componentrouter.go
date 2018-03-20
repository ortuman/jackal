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

	"github.com/jackal-xmpp/stravaganza"
	clusterpb "github.com/ortuman/jackal/cluster/pb"
)

// ComponentRouter defines component router service.
type ComponentRouter interface {
	Route(ctx context.Context, stanza stravaganza.Stanza, componentHost string) error
}

type componentRouter struct {
	cl clusterpb.ComponentRouterClient
}

func (cc *componentRouter) Route(ctx context.Context, stanza stravaganza.Stanza, componentHost string) error {
	_, err := cc.cl.Route(ctx, &clusterpb.ComponentRouteRequest{
		Stanza: stanza.Proto(),
	})
	return err
}
