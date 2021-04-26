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

package extcomponentmanager

import (
	"context"

	"github.com/jackal-xmpp/stravaganza/v2"
	clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"
)

type extComponent struct {
	host string
	conn clusterconnmanager.Conn
}

func newExtComponent(host string, conn clusterconnmanager.Conn) *extComponent {
	return &extComponent{
		host: host,
		conn: conn,
	}
}

func (c *extComponent) Host() string { return c.host }
func (c *extComponent) Name() string { return "" }

func (c *extComponent) ProcessStanza(ctx context.Context, stanza stravaganza.Stanza) error {
	return c.conn.ComponentRouter().Route(ctx, stanza, c.host)
}

func (c *extComponent) Start(_ context.Context) error { return nil }
func (c *extComponent) Stop(_ context.Context) error  { return nil }
