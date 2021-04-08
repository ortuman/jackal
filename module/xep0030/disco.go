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

package xep0030

import (
	"context"
	"errors"
	"sync"

	"github.com/ortuman/jackal/c2s"

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/log"
	discomodel "github.com/ortuman/jackal/model/disco"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
	xmpputil "github.com/ortuman/jackal/util/xmpp"
)

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

var errSubscriptionRequired = errors.New("xep0030: subscription required")

// InfoProvider represents a general entity disco info provider interface.
type InfoProvider interface {
	// Identities returns all identities associated to the provider.
	Identities(ctx context.Context, toJID, fromJID *jid.JID, node string) []discomodel.Identity

	// Items returns all items associated to the provider.
	Items(ctx context.Context, toJID, fromJID *jid.JID, node string) ([]discomodel.Item, error)

	// Features returns all features associated to the provider.
	Features(ctx context.Context, toJID, fromJID *jid.JID, node string) ([]discomodel.Feature, error)

	// Forms returns data forms associated to the provider.
	Forms(ctx context.Context, toJID, fromJID *jid.JID, node string) ([]xep0004.DataForm, error)
}

const (
	// ModuleName represents disco module name.
	ModuleName = "disco"

	// XEPNumber represents disco XEP number.
	XEPNumber = "0030"
)

// Disco represents a disco info (XEP-0030) module type.
type Disco struct {
	router     router.Router
	components components
	rosRep     repository.Roster
	resMng     resourceManager

	mu      sync.RWMutex
	srvProv InfoProvider
	accProv InfoProvider
}

// New returns a new initialized disco module instance.
func New(
	router router.Router,
	components *component.Components,
	rosRep repository.Roster,
	resMng *c2s.ResourceManager,
) *Disco {
	return &Disco{
		router:     router,
		components: components,
		rosRep:     rosRep,
		resMng:     resMng,
	}
}

// Name returns disco module name.
func (d *Disco) Name() string { return ModuleName }

// StreamFeature returns disco stream feature.
func (d *Disco) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns server disco features.
func (d *Disco) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{discoInfoNamespace, discoItemsNamespace}, nil
}

// AccountFeatures returns account disco features.
func (d *Disco) AccountFeatures(_ context.Context) ([]string, error) {
	return []string{discoInfoNamespace, discoItemsNamespace}, nil
}

// MatchesNamespace tells whether namespace matches version module.
func (d *Disco) MatchesNamespace(namespace string, _ bool) bool {
	return namespace == discoInfoNamespace || namespace == discoItemsNamespace
}

// ProcessIQ process a disco info iq.
func (d *Disco) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet():
		return d.getDiscoInfo(ctx, iq)
	case iq.IsSet():
		_, _ = d.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
	}
	return nil
}

// Start starts disco module.
func (d *Disco) Start(_ context.Context) error {
	log.Infow("Started disco module", "xep", XEPNumber)
	return nil
}

// Stop stops disco module.
func (d *Disco) Stop(_ context.Context) error {
	log.Infow("Stopped disco module", "xep", XEPNumber)
	return nil
}

// SetModules set disco modules to be announced on info request.
func (d *Disco) SetModules(mods []module.Module) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.srvProv = newServerProvider(mods, d.components)
	d.accProv = newAccountProvider(mods, d.rosRep, d.resMng)
}

// ServerProvider returns current disco info server provider.
func (d *Disco) ServerProvider() InfoProvider {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.srvProv
}

// AccountProvider returns current disco info account provider.
func (d *Disco) AccountProvider() InfoProvider {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.accProv
}

func (d *Disco) getDiscoInfo(ctx context.Context, iq *stravaganza.IQ) error {
	q := iq.Child("query")
	if q == nil {
		_, _ = d.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	var prov InfoProvider

	d.mu.RLock()
	switch {
	case iq.ToJID().IsServer():
		prov = d.srvProv
	default:
		prov = d.accProv
	}
	d.mu.RUnlock()

	if prov == nil {
		return nil // modules not set
	}
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	node := q.Attribute("node")
	switch q.Attribute(stravaganza.Namespace) {
	case discoInfoNamespace:
		return d.sendDiscoInfo(ctx, prov, toJID, fromJID, node, iq)
	case discoItemsNamespace:
		return d.sendDiscoItems(ctx, prov, toJID, fromJID, node, iq)
	default:
		_, _ = d.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
}

func (d *Disco) sendDiscoInfo(ctx context.Context, prov InfoProvider, toJID, fromJID *jid.JID, node string, iq *stravaganza.IQ) error {
	features, err := prov.Features(ctx, toJID, fromJID, node)
	switch {
	case err == nil:
		break
	case errors.Is(err, errSubscriptionRequired):
		_, _ = d.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.SubscriptionRequired))
		return err
	default:
		_, _ = d.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	sb := stravaganza.NewBuilder("query").
		WithAttribute(stravaganza.Namespace, discoInfoNamespace)

	identities := prov.Identities(ctx, toJID, fromJID, node)
	for _, identity := range identities {
		identityB := stravaganza.NewBuilder("identity")
		identityB.WithAttribute("category", identity.Category)
		if len(identity.Type) > 0 {
			identityB.WithAttribute("type", identity.Type)
		}
		if len(identity.Name) > 0 {
			identityB.WithAttribute("name", identity.Name)
		}
		if len(identity.Lang) > 0 {
			identityB.WithAttribute(stravaganza.Language, identity.Lang)
		}
		sb.WithChild(identityB.Build())
	}
	for _, feature := range features {
		featureB := stravaganza.NewBuilder("feature")
		featureB.WithAttribute("var", feature)
		sb.WithChild(featureB.Build())
	}
	forms, err := prov.Forms(ctx, toJID, fromJID, node)
	if err != nil {
		return err
	}
	for _, form := range forms {
		sb.WithChild(form.Element())
	}
	_, _ = d.router.Route(ctx, xmpputil.MakeResultIQ(iq, sb.Build()))
	return nil
}

func (d *Disco) sendDiscoItems(ctx context.Context, prov InfoProvider, toJID, fromJID *jid.JID, node string, iq *stravaganza.IQ) error {
	items, err := prov.Items(ctx, toJID, fromJID, node)
	switch {
	case err == nil:
		break
	case errors.Is(err, errSubscriptionRequired):
		_, _ = d.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.SubscriptionRequired))
		return err
	default:
		_, _ = d.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	qb := stravaganza.NewBuilder("query").
		WithAttribute(stravaganza.Namespace, discoItemsNamespace)

	for _, item := range items {
		itemB := stravaganza.NewBuilder("item")
		itemB.WithAttribute("jid", item.Jid)
		if len(item.Name) > 0 {
			itemB.WithAttribute("name", item.Name)
		}
		if len(item.Node) > 0 {
			itemB.WithAttribute("node", item.Node)
		}
		qb.WithChild(itemB.Build())
	}
	_, _ = d.router.Route(ctx, xmpputil.MakeResultIQ(iq, qb.Build()))
	return nil
}
