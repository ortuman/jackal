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
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"

	"github.com/jackal-xmpp/stravaganza"
)

const purgeNodes = "purge-nodes"

func (m *Service) purgeNode(ctx context.Context, iq *stravaganza.IQ, purge stravaganza.Element) error {
	if err := m.checkFeature(PurgeNodes, purgeNodes); err != nil {
		return err
	}
	if err := m.authorizeRequestingEntity(ctx, iq, purge, purgeNodes); err != nil {
		return err
	}
	// check if node exists
	node, err := m.fetchNode(ctx, iq, purge)
	if err != nil {
		return err
	}
	if node == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.ItemNotFound))
		return nil
	}
	if err := m.rep.DeleteNodeItems(ctx, node.Host, node.Name); err != nil {
		return err
	}

	// notify node subscribers
	if node.Options.DeliverNotifications && node.Options.NotifyRetract {
		notification := stravaganza.NewBuilder("purge").
			WithAttribute("node", node.Name).
			Build()
		if err := m.notifySubscribers(ctx, node.Host, node.Name, notification, node.Options.NotificationType); err != nil {
			return err
		}
	}

	// send result
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))
	return nil
}
