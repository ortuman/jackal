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

package session

import (
	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/transport"
)

//go:generate moq -out hosts.mock_test.go . hosts
type hosts interface {
	DefaultHostName() string
	IsLocalHost(host string) bool
}

//go:generate moq -out transport.mock_test.go . sessionTransport:transportMock
type sessionTransport interface {
	transport.Transport
}

//go:generate moq -out xmppparser.mock_test.go . xmppParser
type xmppParser interface {
	Parse() (stravaganza.Element, error)
}
