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

package xep0280

import (
	"context"

	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/router"
	xmpputil "github.com/ortuman/jackal/util/xmpp"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/log"
)

const carbonsNamespace = "urn:xmpp:carbons:2"

const (
	// ModuleName represents carbons module name.
	ModuleName = "carbons"

	// XEPNumber represents carbons XEP number.
	XEPNumber = "0280"
)

type Carbons struct {
	router router.Router
}

func New(router router.Router) *Carbons {
	return &Carbons{
		router: router,
	}
}

// Name returns carbons module name.
func (p *Carbons) Name() string { return ModuleName }

// StreamFeature returns carbons module stream feature.
func (p *Carbons) StreamFeature(_ context.Context, _ string) stravaganza.Element { return nil }

// ServerFeatures returns carbons server disco features.
func (p *Carbons) ServerFeatures() []string {
	return []string{carbonsNamespace}
}

// AccountFeatures returns ping account disco features.
func (p *Carbons) AccountFeatures() []string {
	return []string{carbonsNamespace}
}

// Start starts carbons module.
func (p *Carbons) Start(_ context.Context) error {
	log.Infow("Started carbons module", "xep", XEPNumber)
	return nil
}

// Stop stops carbons module.
func (p *Carbons) Stop(_ context.Context) error {
	log.Infow("Stopped carbons module", "xep", XEPNumber)
	return nil
}

// MatchesNamespace tells whether namespace matches carbons module.
func (p *Carbons) MatchesNamespace(namespace string) bool {
	return namespace == carbonsNamespace
}

// ProcessIQ process a carbons iq.
func (p *Carbons) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsSet():
	default:
		_ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
	}
	return nil
}
