// Copyright 2023 The jackal Authors
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

	"github.com/ortuman/jackal/pkg/module/xep0004"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"

	"github.com/jackal-xmpp/stravaganza"
)

const retrieveDefault = "retrieve-default"

func (m *Service) getDefaultConfig(ctx context.Context, iq *stravaganza.IQ) error {
	if err := m.checkFeature(ConfigNode, configNode); err != nil {
		return err
	}
	if !m.cfg.ConfigRetrievalEnabled {
		return newServiceErrorWithFeatureName(errConfigRetrievalDisabled, retrieveDefault)
	}
	x := optionsToForm(m.cfg.DefaultNodeOptions, xep0004.Form)

	// send result IQ
	resIQ := xmpputil.MakeResultIQ(iq,
		stravaganza.NewBuilder("pubsub").
			WithAttribute(stravaganza.Namespace, pubSubOwnerNS()).
			WithChild(
				stravaganza.NewBuilder("default").
					WithChild(x.Element()).
					Build(),
			).
			Build(),
	)
	_, _ = m.router.Route(ctx, resIQ)
	return nil
}
