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

	"github.com/google/uuid"

	"github.com/jackal-xmpp/stravaganza/jid"

	"github.com/jackal-xmpp/stravaganza"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
)

func (m *Service) checkFeature(serviceFeature ServiceFeatures, featureName string) error {
	if !m.cfg.Features.Has(serviceFeature) {
		return newServiceErrorWithFeatureName(errFeatureNotSupported, featureName)
	}
	return nil
}

func (m *Service) fetchNode(ctx context.Context, iq *stravaganza.IQ, cmdElement stravaganza.Element) (*pubsubmodel.Node, error) {
	host, nodeID := getHostAndNodeID(iq, cmdElement)
	if len(nodeID) == 0 {
		return nil, newServiceError(errNodeIDRequired)
	}

	node, err := m.rep.FetchNode(ctx, host, nodeID)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, newServiceError(errNodeNotFound)
	}
	return node, nil
}

func (m *Service) authorizeRequestingEntity(ctx context.Context, iq *stravaganza.IQ, cmdElement stravaganza.Element, feature string) error {
	fromJID := iq.FromJID().ToBareJID().String()

	host, nodeID := getHostAndNodeID(iq, cmdElement)

	return m.hasPrivileges(ctx, fromJID, host, nodeID, feature)
}

func (m *Service) hasPrivileges(ctx context.Context, jid, host, nodeID, feature string) error {
	aff, err := m.rep.FetchNodeAffiliation(ctx, host, nodeID, jid)
	if err != nil {
		return err
	}
	if !hasRequiredPrivileges(aff, feature) {
		return newServiceError(errInsufficientPrivileges)
	}
	return nil
}

func (m *Service) notifySubscribers(ctx context.Context, host, nodeID string, notification stravaganza.Element, notificationType string) error {
	subs, err := m.rep.FetchNodeSubscriptions(ctx, host, nodeID)
	if err != nil {
		return err
	}

	targets := make([]*jid.JID, 0, len(subs))
	for _, sub := range subs {
		if !sub.IsSubscribed() {
			continue
		}
		target, _ := jid.NewWithString(sub.Jid, true)
		targets = append(targets, target)
	}

	// send notification to target JIDs
	for _, target := range targets {
		eventElement := stravaganza.NewBuilder("event").
			WithAttribute(stravaganza.Namespace, pubSubNS("event")).
			WithChild(notification).
			Build()

		msg, _ := stravaganza.NewMessageBuilder().
			WithAttribute(stravaganza.From, host).
			WithAttribute(stravaganza.To, target.String()).
			WithAttribute(stravaganza.ID, uuid.New().String()).
			WithAttribute(stravaganza.Type, notificationType).
			WithChild(eventElement).
			BuildMessage()

		_, _ = m.router.Route(ctx, msg)
	}
	return nil
}

func getHostAndNodeID(iq *stravaganza.IQ, cmdElement stravaganza.Element) (string, string) {
	host := iq.ToJID().ToBareJID().String()
	nodeID := cmdElement.Attribute("node")
	return host, nodeID
}

func hasRequiredPrivileges(affiliation *pubsubmodel.Affiliation, feature string) bool {
	if affiliation == nil {
		return false
	}
	switch feature {
	case configNode, deleteNodes, purgeNodes:
		return affiliation.IsOwner()
	}
	return false
}
