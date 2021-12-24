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

package s2s

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/transport"
)

//go:generate moq -out kv.mock_test.go . kvStorage:kvMock
type kvStorage interface {
	kv.KV
}

//go:generate moq -out router.mock_test.go . globalRouter:routerMock
type globalRouter interface {
	router.Router
}

//go:generate moq -out transport.mock_test.go . s2sTransport:transportMock
type s2sTransport interface {
	transport.Transport
}

//go:generate moq -out netconn.mock_test.go . netConn
type netConn interface {
	net.Conn
}

//go:generate moq -out hosts.mock_test.go . hosts
type hosts interface {
	DefaultHostName() string

	Certificates() []tls.Certificate
	IsLocalHost(host string) bool
}

//go:generate moq -out session.mock_test.go . session
type session interface {
	StreamID() string
	SetFromJID(fromJID *jid.JID)

	Send(ctx context.Context, element stravaganza.Element) error
	Receive() (stravaganza.Element, error)

	OpenStream(ctx context.Context) error
	Close(ctx context.Context) error

	Reset(tr transport.Transport) error
}

//go:generate moq -out components.mock_test.go . components
type components interface {
	IsComponentHost(cHost string) bool
	ProcessStanza(ctx context.Context, stanza stravaganza.Stanza) error
}

//go:generate moq -out modules.mock_test.go . modules
type modules interface {
	IsModuleIQ(iq *stravaganza.IQ) bool
	ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error
}

//go:generate moq -out outprovider.mock_test.go . outProvider
type outProvider interface {
	DialbackSecret() string
	GetOut(ctx context.Context, sender, target string) (stream.S2SOut, error)
	GetDialback(ctx context.Context, sender, target string, params DialbackParams) (stream.S2SDialback, error)
}

//go:generate moq -out s2sin.mock_test.go . s2sIn
type s2sIn interface {
	stream.S2SIn
}

//go:generate moq -out s2sout.mock_test.go . s2sOut
type s2sOut interface {
	stream.S2SOut
	dial(ctx context.Context) error
	start() error
}

//go:generate moq -out s2sdialback.mock_test.go . s2sDialback
type s2sDialback interface {
	stream.S2SDialback
	dial(ctx context.Context) error
	start() error
}
