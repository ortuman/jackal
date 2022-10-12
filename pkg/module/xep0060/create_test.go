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
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/stretchr/testify/require"

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

func TestService_CreateNode(t *testing.T) {
	tcs := []testConfig{
		{
			name: "create node with default options",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub'>
					<create node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return false, nil
				}
				repMock.InTransactionFunc = func(ctx context.Context, fn func(ctx context.Context, tx repository.Transaction) error) error {
					txMock := &txMock{}
					txMock.UpsertNodeFunc = func(ctx context.Context, node *pubsubmodel.Node) error {
						return nil
					}
					txMock.UpsertNodeAffiliationFunc = func(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
						return nil
					}
					txMock.UpsertNodeSubscriptionFunc = func(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
						return nil
					}
					return fn(ctx, txMock)
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ResultType, output[0].Type())
				require.Equal(t, "create1", output[0].ID())
				require.Len(t, output[0].AllChildren(), 0)
			},
		},
		{
			name: "create node with custom options",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create2'>
					<pubsub xmlns='http://jabber.org/protocol/pubsub'>
					  <create node='princely_musings'/>
					  <configure>
						<x xmlns='jabber:x:data' type='submit'>
						  <field var='FORM_TYPE' type='hidden'>
							<value>http://jabber.org/protocol/pubsub#node_config</value>
						  </field>
						  <field var='pubsub#access_model'><value>whitelist</value></field>
						</x>
					  </configure>
					</pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					DefaultNodeOptions: defaultNodeOptions,
					Features:           CreateNodes | CreateAndConfigure,
				}
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return false, nil
				}
				repMock.InTransactionFunc = func(ctx context.Context, fn func(ctx context.Context, tx repository.Transaction) error) error {
					txMock := &txMock{}
					txMock.UpsertNodeFunc = func(ctx context.Context, node *pubsubmodel.Node) error {
						return nil
					}
					txMock.UpsertNodeAffiliationFunc = func(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
						return nil
					}
					txMock.UpsertNodeSubscriptionFunc = func(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
						return nil
					}
					return fn(ctx, txMock)
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ResultType, output[0].Type())
				require.Equal(t, "create2", output[0].ID())
				require.Len(t, output[0].AllChildren(), 0)
			},
		},
		{
			name: "create instant node",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create_instant'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub'>
					<create/>
				  </pubsub>
				</iq>
				`)
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return false, nil
				}
				repMock.InTransactionFunc = func(ctx context.Context, fn func(ctx context.Context, tx repository.Transaction) error) error {
					txMock := &txMock{}
					txMock.UpsertNodeFunc = func(ctx context.Context, node *pubsubmodel.Node) error {
						return nil
					}
					txMock.UpsertNodeAffiliationFunc = func(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
						return nil
					}
					txMock.UpsertNodeSubscriptionFunc = func(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
						return nil
					}
					return fn(ctx, txMock)
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ResultType, output[0].Type())
				require.Equal(t, "create_instant", output[0].ID())

				pubsub := output[0].ChildNamespace("pubsub", pubSubNamespace)
				require.NotNil(t, pubsub)

				create := pubsub.Child("create")
				require.NotNil(t, create)

				require.True(t, len(create.Attribute("node")) > 0)
			},
		},
		{
			name: "failed to create node due feature disabled",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub'>
					<create/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{} // create node feature disabled
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "create1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("feature-not-implemented"))

				unsupportedElem := errorElem.ChildNamespace("unsupported", errorNS())
				require.NotNil(t, unsupportedElem)
				require.Equal(t, createNodes, unsupportedElem.Attribute("feature"))
			},
		},
		{
			name: "failed to create node due to instant nodes disabled",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create_instant'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub'>
					<create/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: CreateNodes,
				}
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return false, nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "create_instant", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("not-acceptable"))
			},
		},
		{
			name: "failed to create node due to not authorized",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub'>
					<create node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: CreateNodes,
					EntityValidator: func(ctx context.Context, jid *jid.JID, featureName string) error {
						return ErrNotAllowed
					},
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "create1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("forbidden"))
			},
		},
		{
			name: "failed to create node due to registration required",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub'>
					<create node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: CreateNodes,
					EntityValidator: func(ctx context.Context, jid *jid.JID, featureName string) error {
						return ErrNotRegisted
					},
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "create1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("registration-required"))
			},
		},
		{
			name: "failed to create node due to conflict",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='set'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='create1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub'>
					<create node='princely_musings'/>
				  </pubsub>
				</iq>
				`)
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return true, nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "create1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("conflict"))
			},
		},
	}
	for _, tCfg := range tcs {
		t.Run(tCfg.name, func(t *testing.T) {
			runServiceTest(t, tCfg)
		})
	}
}
