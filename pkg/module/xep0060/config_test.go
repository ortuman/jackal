// Copyright 2022 The jackal Authors
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

package xep0060

import (
	"testing"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"

	"github.com/stretchr/testify/require"
)

func TestServiceConfig_FeatureList(t *testing.T) {
	// given
	var f ServiceFeatures

	f |= AutoCreate
	f |= AutoSubscribe
	f |= ConfigNode
	f |= CreateAndConfigure
	f |= CreateNodes
	f |= DeleteNodes
	f |= FilteredNotifications
	f |= InstantNodes
	f |= PersistentItems
	f |= RetrieveAffiliations
	f |= RetrieveSubscriptions
	f |= RetrieveItems

	cfg := &ServiceConfig{
		Features: f,
		DefaultNodeOptions: &pubsubmodel.Options{
			AccessModel: pubsubmodel.NodeAccessModel2String[pubsubmodel.NodeAccessModel_NAM_OPEN],
		},
	}

	// when
	featureList := cfg.FeatureList()

	// then
	require.Equal(t, []string{
		"http://jabber.org/protocol/pubsub#open",
		"http://jabber.org/protocol/pubsub#auto-create",
		"http://jabber.org/protocol/pubsub#auto-subscribe",
		"http://jabber.org/protocol/pubsub#config-node",
		"http://jabber.org/protocol/pubsub#create-and-configure",
		"http://jabber.org/protocol/pubsub#create-nodes",
		"http://jabber.org/protocol/pubsub#delete-nodes",
		"http://jabber.org/protocol/pubsub#filtered-notifications",
		"http://jabber.org/protocol/pubsub#instant-nodes",
		"http://jabber.org/protocol/pubsub#persistent-items",
		"http://jabber.org/protocol/pubsub#retrieve-affiliations",
		"http://jabber.org/protocol/pubsub#retrieve-subscriptions",
		"http://jabber.org/protocol/pubsub#retrieve-items",
		"http://jabber.org/protocol/pubsub#publish",
		"http://jabber.org/protocol/pubsub#subscribe",
	}, featureList)
}
