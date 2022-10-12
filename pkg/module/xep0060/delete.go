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

	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/pkg/storage/repository"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"

	"github.com/jackal-xmpp/stravaganza"
)

const deleteNodes = "delete-nodes"

func (m *Service) deleteNode(ctx context.Context, iq *stravaganza.IQ, del stravaganza.Element) error {
	if err := m.checkFeature(DeleteNodes, deleteNodes); err != nil {
		return err
	}
	if err := m.authorizeRequestingEntity(ctx, iq, del, configNode); err != nil {
		return err
	}
	// check if node exists
	node, err := m.fetchNode(ctx, iq, del)
	if err != nil {
		return err
	}
	if node == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.ItemNotFound))
		return nil
	}
	// get node identifier
	host := node.Host
	nodeID := node.Name

	// delete node
	err = m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if err := tx.DeleteNodeItems(ctx, host, nodeID); err != nil {
			return err
		}
		if err := tx.DeleteNodeSubscriptions(ctx, host, nodeID); err != nil {
			return err
		}
		if err := tx.DeleteNodeAffiliations(ctx, host, nodeID); err != nil {
			return err
		}
		return tx.DeleteNode(ctx, host, nodeID)
	})
	if err != nil {
		return err
	}

	// notify node subscribers
	if node.Options.DeliverNotifications && node.Options.NotifyDelete {
		notification := stravaganza.NewBuilder("delete").
			WithAttribute("node", nodeID).
			Build()
		if err := m.notifySubscribers(ctx, host, nodeID, notification, node.Options.NotificationType); err != nil {
			return err
		}
	}

	// send result
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))
	return nil
}
