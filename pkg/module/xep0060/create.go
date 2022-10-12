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
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/ortuman/jackal/pkg/module/xep0004"
	"github.com/ortuman/jackal/pkg/storage/repository"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const createNodes = "create-nodes"

func (m *Service) createNode(ctx context.Context, iq *stravaganza.IQ, create, configure stravaganza.Element) error {
	if err := m.checkFeature(CreateNodes, createNodes); err != nil {
		return err
	}
	// validate requesting entity
	fromJID := iq.FromJID().ToBareJID()

	if entityValidator := m.cfg.EntityValidator; entityValidator != nil {
		if err := entityValidator(ctx, fromJID, createNodes); err != nil {
			return err
		}
	}

	// get node identifier
	host, nodeID := getHostAndNodeID(iq, create)

	var isInstantNode bool
	if len(nodeID) == 0 {
		if !m.cfg.Features.Has(InstantNodes) {
			_, _ = m.router.Route(ctx, nodeIDRequiredError(iq, stanzaerror.NotAcceptable))
			return nil
		}
		nodeID = uuid.New().String()
		isInstantNode = true
	}

	// check if node already exists
	exists, err := m.rep.NodeExists(ctx, host, nodeID)
	if err != nil {
		return err
	}
	if exists {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Conflict))
		return nil
	}

	opts := m.cfg.DefaultNodeOptions
	if configure != nil && m.cfg.Features.Has(CreateAndConfigure) {
		if x := configure.ChildNamespace("x", xep0004.FormNamespace); x != nil {
			opts, err = formToOptions(m.cfg.DefaultNodeOptions, x)
			if err != nil {
				_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
				return nil
			}
		}
	}
	err = m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if err := tx.UpsertNode(ctx, &pubsubmodel.Node{
			Host:    host,
			Name:    nodeID,
			Type:    pubsubmodel.NodeType_NT_LEAF,
			Options: opts,
		}); err != nil {
			return err
		}
		if err := tx.UpsertNodeAffiliation(ctx, &pubsubmodel.Affiliation{
			Jid:   fromJID.String(),
			State: pubsubmodel.AffiliationState_AFF_OWNER,
		}, host, nodeID); err != nil {
			return err
		}
		return tx.UpsertNodeSubscription(ctx, &pubsubmodel.Subscription{
			Jid:   fromJID.String(),
			State: pubsubmodel.SubscriptionState_SUB_SUBSCRIBED,
		}, host, nodeID)
	})
	if err != nil {
		return err
	}

	// send result IQ
	var resChild stravaganza.Element
	if isInstantNode {
		resChild = stravaganza.NewBuilder("pubsub").
			WithAttribute(stravaganza.Namespace, pubSubNamespace).
			WithChild(
				stravaganza.NewBuilder("create").
					WithAttribute("node", nodeID).
					Build(),
			).Build()
	}
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, resChild))
	return nil
}
