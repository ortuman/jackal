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
	"testing"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"

	"github.com/stretchr/testify/require"

	"github.com/jackal-xmpp/stravaganza"
)

func TestService_PurgeNode(t *testing.T) {
	tcs := []testConfig{
		{
			name: "purge all node items",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='purge1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<purge node='princely_musings'/>
				  </pubsub>
				</iq>
			`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: PurgeNodes,
				}
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.FetchNodeAffiliationFunc = func(ctx context.Context, jid string, host string, name string) (*pubsubmodel.Affiliation, error) {
					return &pubsubmodel.Affiliation{
						Jid:   "hamlet@denmark.lit",
						State: pubsubmodel.AffiliationState_AFF_OWNER,
					}, nil
				}
				repMock.FetchNodeFunc = func(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
					return &pubsubmodel.Node{
						Host:    "pubsub.shakespeare.lit",
						Name:    "princely_musings",
						Options: defaultNodeOptions,
					}, nil
				}
				repMock.DeleteNodeItemsFunc = func(ctx context.Context, host, name string) error {
					return nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ResultType, output[0].Type())
				require.Equal(t, "purge1", output[0].ID())
			},
		},
		{
			name: "service does not support purge",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='purge1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<purge node='princely_musings'/>
				  </pubsub>
				</iq>
			`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{} // purge node items disabled
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "purge1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("feature-not-implemented"))

				unsupportedElem := errorElem.ChildNamespace("unsupported", errorNS())
				require.NotNil(t, unsupportedElem)
				require.Equal(t, purgeNodes, unsupportedElem.Attribute("feature"))
			},
		},
		{
			name: "failed to purge node items due to not authorized",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='purge1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<purge node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: PurgeNodes,
				}
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.FetchNodeAffiliationFunc = func(ctx context.Context, jid string, host string, name string) (*pubsubmodel.Affiliation, error) {
					return &pubsubmodel.Affiliation{
						Jid:   "hamlet@denmark.lit",
						State: pubsubmodel.AffiliationState_AFF_PUBLISHER,
					}, nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "purge1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("forbidden"))
			},
		},
	}
	for _, tCfg := range tcs {
		t.Run(tCfg.name, func(t *testing.T) {
			runServiceTest(t, tCfg)
		})
	}
}
