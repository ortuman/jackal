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

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
	xmpputil "github.com/ortuman/jackal/util/xmpp"
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
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New returns a new initialized VCard instance.
func New(rep repository.Repository, router router.Router, sn *sonar.Sonar) *VCard {
	return &VCard{
		rep:    rep,
		router: router,
		sn:     sn,
	}
}

// Name returns vCard module name.
func (v *VCard) Name() string { return ModuleName }

// StreamFeature returns vCard module stream feature.
func (v *VCard) StreamFeature() stravaganza.Element { return nil }

// ServerFeatures returns vCard server disco features.
func (v *VCard) ServerFeatures() []string {
	return []string{vCardNamespace}
}

// AccountFeatures returns vCard account disco features.
func (v *VCard) AccountFeatures() []string {
	return []string{vCardNamespace}
}

// MatchesNamespace tells whether namespace matches vCard module.
func (v *VCard) MatchesNamespace(namespace string) bool {
	return namespace == vCardNamespace
}

// ProcessIQ process a vCard iq.
func (v *VCard) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet():
		return v.getVCard(ctx, iq)
	case iq.IsSet():
		return v.setVCard(ctx, iq)
	}
	return nil
}

// Start starts vCard module.
func (v *VCard) Start(_ context.Context) error {
	v.subs = append(v.subs, v.sn.Subscribe(event.UserDeleted, v.onUserDeleted))

	log.Infow("Started vCard module", "xep", XEPNumber)
	return nil
}

// Stop stops vCard module.
func (v *VCard) Stop(_ context.Context) error {
	for _, sub := range v.subs {
		v.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped vCard module", "xep", XEPNumber)
	return nil
}

func (v *VCard) onUserDeleted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.UserEventInfo)
	return v.rep.DeleteVCard(ctx, inf.Username)
}

func (v *VCard) getVCard(ctx context.Context, iq *stravaganza.IQ) error {
	vc := iq.ChildNamespace("vCard", vCardNamespace)
	if vc == nil || vc.ChildrenCount() > 0 {
		_ = v.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	toJID := iq.ToJID()
	vCard, err := v.rep.FetchVCard(ctx, toJID.Node())
	if err != nil {
		_ = v.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
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

	_ = v.router.Route(ctx, resIQ)
	return nil
}

func (v *VCard) setVCard(ctx context.Context, iq *stravaganza.IQ) error {
	vc := iq.ChildNamespace("vCard", vCardNamespace)
	if vc == nil {
		_ = v.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	allowed := toJID.IsServer() || (toJID.Node() == fromJID.Node())
	if !allowed {
		_ = v.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil
	}
	err := v.rep.UpsertVCard(ctx, vc, toJID.Node())
	if err != nil {
		_ = v.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err

	}
	log.Infow("Saved vCard", "vcard", toJID.Node(), "xep", XEPNumber)

	_ = v.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))

	// post vCard update event
	return v.sn.Post(
		ctx,
		sonar.NewEventBuilder(event.VCardUpdated).
			WithInfo(&event.VCardEventInfo{
				Username: toJID.Node(),
			}).Build(),
	)
}
