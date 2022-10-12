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

	"github.com/jackal-xmpp/stravaganza"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/ortuman/jackal/pkg/storage/repository"
	"github.com/stretchr/testify/require"
)

func TestService_DeleteNode(t *testing.T) {
	tcs := []testConfig{
		{
			name: "delete node",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='delete1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<delete node='princely_musings'/>
				  </pubsub>
				</iq>
			`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: DeleteNodes,
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
				repMock.FetchNodeSubscriptionsFunc = func(ctx context.Context, host, node string) ([]*pubsubmodel.Subscription, error) {
					return []*pubsubmodel.Subscription{
						{
							Jid:   "francisco@denmark.lit",
							State: pubsubmodel.SubscriptionState_SUB_SUBSCRIBED,
						},
					}, nil
				}

				tx := txMock{}
				tx.DeleteNodeItemsFunc = func(ctx context.Context, host, name string) error {
					return nil
				}
				tx.DeleteNodeSubscriptionsFunc = func(ctx context.Context, host string, name string) error {
					return nil
				}
				tx.DeleteNodeAffiliationsFunc = func(ctx context.Context, host string, name string) error {
					return nil
				}
				tx.DeleteNodeFunc = func(ctx context.Context, host string, name string) error {
					return nil
				}

				repMock.InTransactionFunc = func(ctx context.Context, fn func(ctx context.Context, tx repository.Transaction) error) error {
					return fn(ctx, &tx)
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 2)

				require.Equal(t, stravaganza.MessageName, output[0].Name())
				require.Equal(t, "francisco@denmark.lit", output[0].ToJID().String())

				eventElem := output[0].ChildNamespace("event", pubSubNS("event"))
				require.NotNil(t, eventElem)

				deleteElem := eventElem.Child("delete")
				require.NotNil(t, deleteElem)
				require.Equal(t, "princely_musings", deleteElem.Attribute("node"))

				require.Equal(t, stravaganza.IQName, output[1].Name())
				require.Equal(t, stravaganza.ResultType, output[1].Type())
				require.Equal(t, "delete1", output[1].ID())
			},
		},
		{
			name: "failed to delete node due to not authorized",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='delete1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<delete node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: DeleteNodes,
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
				require.Equal(t, "delete1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("forbidden"))
			},
		},
		{
			name: "failed to delete node due to feature disabled",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='delete1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<delete node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{} // delete node feature disabled
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "delete1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("feature-not-implemented"))

				unsupportedElem := errorElem.ChildNamespace("unsupported", errorNS())
				require.NotNil(t, unsupportedElem)
				require.Equal(t, deleteNodes, unsupportedElem.Attribute("feature"))
			},
		},
		{
			name: "failed to delete node due to non-existing node",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='delete1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<delete node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: DeleteNodes,
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
					return nil, nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "delete1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("item-not-found"))
			},
		},
	}
	for _, tCfg := range tcs {
		t.Run(tCfg.name, func(t *testing.T) {
			runServiceTest(t, tCfg)
		})
	}
}
