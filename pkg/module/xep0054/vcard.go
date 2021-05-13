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

package xep0054

import (
	"context"

	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/ortuman/jackal/pkg/event"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/repository"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const vCardNamespace = "vcard-temp"

const (
	// ModuleName represents vCard module name.
	ModuleName = "vcard"

	// XEPNumber represents vCard XEP number.
	XEPNumber = "0054"
)

// VCard represents a vCard (XEP-0054) module type.
type VCard struct {
	rep    repository.VCard
	router router.Router
	mh     *module.Hooks
}

// New returns a new initialized VCard instance.
func New(router router.Router, rep repository.Repository, mh *module.Hooks) *VCard {
	return &VCard{
		router: router,
		rep:    rep,
		mh:     mh,
	}
}

// Name returns vCard module name.
func (m *VCard) Name() string { return ModuleName }

// StreamFeature returns vCard module stream feature.
func (m *VCard) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns vCard server disco features.
func (m *VCard) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{vCardNamespace}, nil
}

// AccountFeatures returns vCard account disco features.
func (m *VCard) AccountFeatures(_ context.Context) ([]string, error) {
	return []string{vCardNamespace}, nil
}

// MatchesNamespace tells whether namespace matches vCard module.
func (m *VCard) MatchesNamespace(namespace string, _ bool) bool {
	return namespace == vCardNamespace
}

// ProcessIQ process a vCard iq.
func (m *VCard) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet():
		return m.getVCard(ctx, iq)
	case iq.IsSet():
		return m.setVCard(ctx, iq)
	}
	return nil
}

// Start starts vCard module.
func (m *VCard) Start(_ context.Context) error {
	m.mh.AddHook(event.UserDeleted, m.onUserDeleted, module.DefaultPriority)

	log.Infow("Started vCard module", "xep", XEPNumber)
	return nil
}

// Stop stops vCard module.
func (m *VCard) Stop(_ context.Context) error {
	m.mh.RemoveHook(event.UserDeleted, m.onUserDeleted)

	log.Infow("Stopped vCard module", "xep", XEPNumber)
	return nil
}

func (m *VCard) onUserDeleted(ctx context.Context, execCtx *module.HookExecutionContext) (halt bool, err error) {
	inf := execCtx.Info.(*event.UserEventInfo)
	return false, m.rep.DeleteVCard(ctx, inf.Username)
}

func (m *VCard) getVCard(ctx context.Context, iq *stravaganza.IQ) error {
	vc := iq.ChildNamespace("vCard", vCardNamespace)
	if vc == nil || vc.ChildrenCount() > 0 {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	toJID := iq.ToJID()
	vCard, err := m.rep.FetchVCard(ctx, toJID.Node())
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	var resIQ *stravaganza.IQ
	if vCard != nil {
		resIQ = xmpputil.MakeResultIQ(iq, vCard)
	} else {
		// empty vCard
		resIQ = xmpputil.MakeResultIQ(iq, stravaganza.NewBuilder("vCard").
			WithAttribute(stravaganza.Namespace, vCardNamespace).
			Build())
	}
	log.Infow("Fetched vCard", "username", iq.FromJID().Node(), "vcard", toJID.Node(), "xep", XEPNumber)

	_, _ = m.router.Route(ctx, resIQ)

	// run vCard fetched hook
	_, err = m.mh.Run(ctx, event.VCardFetched, &module.HookExecutionContext{
		Info: &event.VCardEventInfo{
			Username: toJID.Node(),
			VCard:    vCard,
		},
		Sender: m,
	})
	return err
}

func (m *VCard) setVCard(ctx context.Context, iq *stravaganza.IQ) error {
	vCard := iq.ChildNamespace("vCard", vCardNamespace)
	if vCard == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	allowed := toJID.IsServer() || (toJID.Node() == fromJID.Node())
	if !allowed {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil
	}
	err := m.rep.UpsertVCard(ctx, vCard, toJID.Node())
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	log.Infow("Saved vCard", "vcard", toJID.Node(), "xep", XEPNumber)

	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))

	// run vCard updated hook
	_, err = m.mh.Run(ctx, event.VCardUpdated, &module.HookExecutionContext{
		Info: &event.VCardEventInfo{
			Username: toJID.Node(),
			VCard:    vCard,
		},
		Sender: m,
	})
	return err
}
