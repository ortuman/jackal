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
	"fmt"
	"sort"

	"github.com/ortuman/jackal/module/xep0004"

	"github.com/jackal-xmpp/stravaganza/jid"
	discomodel "github.com/ortuman/jackal/model/disco"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/repository"
)

type accountProvider struct {
	mods   []module.Module
	rosRep repository.Roster
	resMng resourceManager
}

func newAccountProvider(
	mods []module.Module,
	rosRep repository.Roster,
	resMng resourceManager,
) *accountProvider {
	return &accountProvider{
		mods:   mods,
		rosRep: rosRep,
		resMng: resMng,
	}
}

func (p *accountProvider) Identities(_ context.Context, _, _ *jid.JID, _ string) []discomodel.Identity {
	return []discomodel.Identity{{Type: "registered", Category: "account"}}
}

func (p *accountProvider) Items(ctx context.Context, toJID, fromJID *jid.JID, _ string) ([]discomodel.Item, error) {
	if err := p.checkIfSubscribedTo(ctx, toJID, fromJID); err != nil {
		return nil, err
	}
	rss, err := p.resMng.GetResources(ctx, toJID.Node())
	if err != nil {
		return nil, err
	}
	var items []discomodel.Item
	for _, res := range rss {
		items = append(items, discomodel.Item{
			Name: res.JID.Node(),
			Jid:  res.JID.String(),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Jid < items[j].Jid })
	return items, nil
}

func (p *accountProvider) Features(ctx context.Context, toJID, fromJID *jid.JID, _ string) ([]discomodel.Feature, error) {
	if err := p.checkIfSubscribedTo(ctx, toJID, fromJID); err != nil {
		return nil, err
	}
	var features []discomodel.Feature
	for _, mod := range p.mods {
		features = append(features, mod.AccountFeatures()...)
	}
	sort.Slice(features, func(i, j int) bool { return features[i] < features[j] })
	return features, nil
}

func (p *accountProvider) Forms(ctx context.Context, toJID, fromJID *jid.JID, node string) ([]xep0004.DataForm, error) {
	return nil, nil
}

func (p *accountProvider) checkIfSubscribedTo(ctx context.Context, toJID, fromJID *jid.JID) error {
	isSubscribed, err := p.isSubscribedTo(ctx, toJID, fromJID)
	if err != nil {
		return err
	}
	if !isSubscribed {
		return fmt.Errorf("%w: from: %s, to: %s", errSubscriptionRequired, fromJID, toJID)
	}
	return nil
}

func (p *accountProvider) isSubscribedTo(ctx context.Context, contact *jid.JID, userJID *jid.JID) (ok bool, err error) {
	if contact.MatchesWithOptions(userJID, jid.MatchesBare) {
		return true, nil
	}
	ri, err := p.rosRep.FetchRosterItem(ctx, userJID.Node(), contact.ToBareJID().String())
	if err != nil {
		return false, err
	}
	if ri == nil {
		return false, nil
	}
	isSubscribed := ri.Subscription == rostermodel.To || ri.Subscription == rostermodel.Both
	return isSubscribed, nil
}
