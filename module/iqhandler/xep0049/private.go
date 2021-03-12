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

package xep0049

import (
	"context"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
)

const privateNamespace = "jabber:iq:private"

const (
	// ModuleName represents private module name.
	ModuleName = "private"

	// XEPNumber represents private XEP number.
	XEPNumber = "0049"
)

// Private represents a private (XEP-0049) module type.
type Private struct {
	rep    repository.Private
	router router.Router
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New returns a new initialized Private instance.
func New(rep repository.Private, router router.Router, sn *sonar.Sonar) *Private {
	return &Private{
		rep:    rep,
		router: router,
		sn:     sn,
	}
}

// Name returns private module name.
func (p *Private) Name() string { return ModuleName }

// StreamFeature returns private module stream feature.
func (p *Private) StreamFeature(_ context.Context, _ string) stravaganza.Element { return nil }

// ServerFeatures returns private server disco features.
func (p *Private) ServerFeatures() []string { return nil }

// AccountFeatures returns private account disco features.
func (p *Private) AccountFeatures() []string { return nil }

// MatchesNamespace tells whether namespace matches private module.
func (p *Private) MatchesNamespace(namespace string) bool {
	return namespace == privateNamespace
}

// ProcessIQ process a private iq.
func (p *Private) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	return nil
}

// Start starts private module.
func (p *Private) Start(_ context.Context) error {
	p.subs = append(p.subs, p.sn.Subscribe(event.UserDeleted, p.onUserDeleted))

	log.Infow("Started private module", "xep", XEPNumber)
	return nil
}

// Stop stops private module.
func (p *Private) Stop(_ context.Context) error {
	for _, sub := range p.subs {
		p.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped private module", "xep", XEPNumber)
	return nil
}

func (p *Private) onUserDeleted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.UserEventInfo)
	return p.rep.DeletePrivates(ctx, inf.Username)
}
