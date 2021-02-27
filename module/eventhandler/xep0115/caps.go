// Copyright 2021 The jackal Authors
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

package xep0115

import (
	"context"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
)

const capabilitiesFeature = "http://jabber.org/protocol/caps"

const (
	// ModuleName represents entity capabilities module name.
	ModuleName = "caps"

	// XEPNumber represents entity capabilities XEP number.
	XEPNumber = "0115"
)

// Capabilities represents entity capabilities module type.
type Capabilities struct {
	router router.Router
	rep    repository.Capabilities
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New creates and initializes a new Capabilities instance.
func New(
	router router.Router,
	rep repository.Capabilities,
	sn *sonar.Sonar,
) *Capabilities {
	return &Capabilities{
		router: router,
		rep:    rep,
		sn:     sn,
	}
}

// Name returns entity capabilities module name.
func (m *Capabilities) Name() string { return ModuleName }

// ServerFeatures returns entity capabilities module server disco features.
func (m *Capabilities) ServerFeatures() []string { return []string{capabilitiesFeature} }

// AccountFeatures returns entity capabilities module account disco features.
func (m *Capabilities) AccountFeatures() []string { return []string{capabilitiesFeature} }

// Start starts entity capabilities module.
func (m *Capabilities) Start(_ context.Context) error {
	log.Infow("Started capabilities module", "xep", XEPNumber)
	return nil
}

// Stop stops entity capabilities module.
func (m *Capabilities) Stop(_ context.Context) error {
	for _, sub := range m.subs {
		m.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped capabilities module", "xep", XEPNumber)
	return nil
}
