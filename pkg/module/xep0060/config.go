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
	"context"
	"errors"

	"github.com/jackal-xmpp/stravaganza/jid"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
)

// RequestingEntityValidator is a function type used to validate requesting entity for a given feature.
type RequestingEntityValidator func(ctx context.Context, jid *jid.JID, featureName string) error

var (
	// ErrNotRegisted is returned when registration is required for requesting entity.
	ErrNotRegisted = errors.New("not registered")

	// ErrNotAllowed is returned when requesting entity is not allowed to perform the requested action.
	ErrNotAllowed = errors.New("not allowed")
)

// ServiceFeatures represents a PubSub service features.
type ServiceFeatures uint64

// PubSub service features.
// Ref: https://xmpp.org/extensions/xep-0060.html#features
const (
	AutoCreate            ServiceFeatures = 1 << 0
	AutoSubscribe                         = 1 << 1
	ConfigNode                            = 1 << 2
	CreateAndConfigure                    = 1 << 3
	CreateNodes                           = 1 << 4
	DeleteNodes                           = 1 << 5
	PurgeNodes                            = 1 << 6
	FilteredNotifications                 = 1 << 7
	InstantNodes                          = 1 << 8
	PersistentItems                       = 1 << 9
	RetrieveAffiliations                  = 1 << 10
	RetrieveSubscriptions                 = 1 << 11
	RetrieveItems                         = 1 << 12
)

// Has returns true if f features are included in sf.
func (sf ServiceFeatures) Has(f ServiceFeatures) bool {
	return sf&f > 0
}

// ServiceConfig is the PubSub service configuration.
type ServiceConfig struct {
	// DefaultNodeOptions is the default node options.
	DefaultNodeOptions *pubsubmodel.Options

	// Features is the list of PubSub service features.
	Features ServiceFeatures

	// EntityValidator is the requesting entity validator function.
	EntityValidator RequestingEntityValidator

	// ConfigRetrievalEnabled tells whether default node configuration can be retrieved.
	ConfigRetrievalEnabled bool
}

// FeatureList returns a list of PubSub service features.
func (cfg ServiceConfig) FeatureList() []string {
	var ret []string

	ret = append(ret, pubSubNS(cfg.DefaultNodeOptions.AccessModel))

	if cfg.Features.Has(AutoCreate) {
		ret = append(ret, pubSubNS("auto-create"))
	}
	if cfg.Features.Has(AutoSubscribe) {
		ret = append(ret, pubSubNS("auto-subscribe"))
	}
	if cfg.Features.Has(ConfigNode) {
		ret = append(ret, pubSubNS("config-node"))
	}
	if cfg.Features.Has(CreateAndConfigure) {
		ret = append(ret, pubSubNS("create-and-configure"))
	}
	if cfg.Features.Has(CreateNodes) {
		ret = append(ret, pubSubNS("create-nodes"))
	}
	if cfg.Features.Has(DeleteNodes) {
		ret = append(ret, pubSubNS("delete-nodes"))
	}
	if cfg.Features.Has(FilteredNotifications) {
		ret = append(ret, pubSubNS("filtered-notifications"))
	}
	if cfg.Features.Has(InstantNodes) {
		ret = append(ret, pubSubNS("instant-nodes"))
	}
	if cfg.Features.Has(PersistentItems) {
		ret = append(ret, pubSubNS("persistent-items"))
	}
	if cfg.Features.Has(RetrieveAffiliations) {
		ret = append(ret, pubSubNS("retrieve-affiliations"))
	}
	if cfg.Features.Has(RetrieveSubscriptions) {
		ret = append(ret, pubSubNS("retrieve-subscriptions"))
	}
	if cfg.Features.Has(RetrieveItems) {
		ret = append(ret, pubSubNS("retrieve-items"))
	}
	ret = append(ret, pubSubNS("publish"))
	ret = append(ret, pubSubNS("subscribe"))
	return ret
}
