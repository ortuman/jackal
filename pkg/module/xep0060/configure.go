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

	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"

	"github.com/ortuman/jackal/pkg/module/xep0004"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const configNode = "config-node"

func (m *Service) getNodeConfig(ctx context.Context, iq *stravaganza.IQ, configure stravaganza.Element) error {
	if err := m.checkFeature(ConfigNode, configNode); err != nil {
		return err
	}
	if err := m.authorizeRequestingEntity(ctx, iq, configure, configNode); err != nil {
		return err
	}
	node, err := m.fetchNode(ctx, iq, configure)
	if err != nil {
		return err
	}
	// convert node options to form
	x := optionsToForm(node.Options, xep0004.Form)

	// send result IQ
	_, _ = m.router.Route(ctx,
		xmpputil.MakeResultIQ(iq,
			stravaganza.NewBuilder("pubsub").
				WithAttribute(stravaganza.Namespace, pubSubOwnerNS()).
				WithChild(
					stravaganza.NewBuilder("configure").
						WithAttribute("node", node.Name).
						WithChild(x.Element()).
						Build(),
				).
				Build(),
		),
	)
	return nil
}

func (m *Service) setNodeConfig(ctx context.Context, iq *stravaganza.IQ, configure stravaganza.Element) error {
	if err := m.checkFeature(ConfigNode, configNode); err != nil {
		return err
	}
	if err := m.authorizeRequestingEntity(ctx, iq, configure, configNode); err != nil {
		return err
	}
	node, err := m.fetchNode(ctx, iq, configure)
	if err != nil {
		return err
	}
	x := configure.ChildNamespace("x", xep0004.FormNamespace)
	if x == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	if x.Attribute("type") == xep0004.Cancel {
		_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))
		return nil
	}

	// convert form to node options taking current ones as defaults
	opts, err := formToOptions(node.Options, x)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	node.Options = opts

	// update node options
	if err := m.rep.UpsertNode(ctx, node); err != nil {
		return err
	}

	// notify node subscribers
	if node.Options.DeliverNotifications && node.Options.NotifyConfig {
		notificationBuilder := stravaganza.NewBuilder("configuration").
			WithAttribute("node", node.Name)

		if node.Options.DeliverPayloads {
			notificationBuilder.WithChild(
				optionsToForm(node.Options, xep0004.Result).Element(),
			)
		}
		host := node.Host
		nodeID := node.Name
		notification := notificationBuilder.Build()

		if err := m.notifySubscribers(ctx, host, nodeID, notification, node.Options.NotificationType); err != nil {
			return err
		}
	}

	// send result IQ
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))
	return nil
}
