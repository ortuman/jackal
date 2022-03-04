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

package xep0114

import (
	"context"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/transport"
)

//go:generate moq -out transport.mock_test.go . componentTransport:transportMock
type componentTransport interface {
	transport.Transport
}

//go:generate moq -out router.mock_test.go . globalRouter:routerMock
type globalRouter interface {
	router.Router
}

//go:generate moq -out session.mock_test.go . session
type session interface {
	StreamID() string
	SetFromJID(ssJID *jid.JID)

	Send(ctx context.Context, element stravaganza.Element) error
	Receive() (stravaganza.Element, error)

	OpenComponent(ctx context.Context) error
	Close(ctx context.Context) error

	Reset(tr transport.Transport) error
}

//go:generate moq -out components.mock_test.go . components
type components interface {
	IsComponentHost(cHost string) bool

	RegisterComponent(ctx context.Context, compo component.Component) error
	UnregisterComponent(ctx context.Context, cHost string) error
}

//go:generate moq -out extcomponentmanager.mock_test.go . externalComponentManager
type externalComponentManager interface {
	RegisterComponentHost(ctx context.Context, cHost string) error
	UnregisterComponentHost(ctx context.Context, cHost string) error
}
