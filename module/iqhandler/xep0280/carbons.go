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
	"fmt"
	"strconv"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	xmpputil "github.com/ortuman/jackal/util/xmpp"
)

const (
	carbonsEnabledCtxKey = "carbons:enabled"

	carbonsNamespace = "urn:xmpp:carbons:2"
)

const (
	// ModuleName represents carbons module name.
	ModuleName = "carbons"

	// XEPNumber represents carbons XEP number.
	XEPNumber = "0280"
)

// Carbons represents carbons (XEP-0280) module type.
type Carbons struct {
	hosts  *host.Hosts
	router router.Router
}

// New returns a new initialized carbons instance.
func New(hosts *host.Hosts, router router.Router) *Carbons {
	return &Carbons{
		hosts:  hosts,
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
		return p.processIQ(ctx, iq)
	default:
		_ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
	}
	return nil
}

func (p *Carbons) processIQ(ctx context.Context, iq *stravaganza.IQ) error {
	fromJID := iq.FromJID()
	if !p.hosts.IsLocalHost(fromJID.Domain()) {
		_ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.NotAllowed))
		return nil
	}
	switch {
	case iq.ChildNamespace("enable", carbonsNamespace) != nil:
		return p.setCarbonsEnabled(ctx, fromJID.Node(), fromJID.Resource(), true)
	case iq.ChildNamespace("disable", carbonsNamespace) != nil:
		return p.setCarbonsEnabled(ctx, fromJID.Node(), fromJID.Resource(), false)
	default:
		_ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
}

func (p *Carbons) setCarbonsEnabled(ctx context.Context, username, resource string, enabled bool) error {
	stm := p.router.C2S().LocalStream(username, resource)
	if stm == nil {
		return errStreamNotFound(username, resource)
	}
	return stm.SetValue(ctx, carbonsEnabledCtxKey, strconv.FormatBool(enabled))
}

func errStreamNotFound(username, resource string) error {
	return fmt.Errorf("xep0280: local stream not found: %s/%s", username, resource)
}
