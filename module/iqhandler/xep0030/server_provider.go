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
	"sort"

	"github.com/ortuman/jackal/module/xep0004"

	"github.com/jackal-xmpp/stravaganza/jid"
	discomodel "github.com/ortuman/jackal/model/disco"
	"github.com/ortuman/jackal/module"
)

type serverProvider struct {
	mods  []module.Module
	comps components
}

func newServerProvider(
	mods []module.Module,
	comps components,
) *serverProvider {
	return &serverProvider{
		mods:  mods,
		comps: comps,
	}
}

func (p *serverProvider) Identities(_ context.Context, _, _ *jid.JID, _ string) []discomodel.Identity {
	return []discomodel.Identity{{Type: "im", Category: "server", Name: "jackal"}}
}

func (p *serverProvider) Items(_ context.Context, _, _ *jid.JID, _ string) ([]discomodel.Item, error) {
	var items []discomodel.Item
	for _, comp := range p.comps.AllComponents() {
		items = append(items, discomodel.Item{
			Jid:  comp.Host(),
			Name: comp.Name(),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Jid < items[j].Jid })
	return items, nil
}

func (p *serverProvider) Features(ctx context.Context, _, _ *jid.JID, _ string) ([]discomodel.Feature, error) {
	var features []discomodel.Feature
	for _, mod := range p.mods {
		srvFeatures, err := mod.ServerFeatures(ctx)
		if err != nil {
			return nil, err
		}
		features = append(features, srvFeatures...)
	}
	sort.Slice(features, func(i, j int) bool { return features[i] < features[j] })
	return features, nil
}

func (p *serverProvider) Forms(ctx context.Context, toJID, fromJID *jid.JID, node string) ([]xep0004.DataForm, error) {
	return nil, nil
}
