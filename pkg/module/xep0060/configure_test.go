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

	"github.com/jackal-xmpp/stravaganza"
	"github.com/stretchr/testify/require"
)

func TestService_ConfigureNode(t *testing.T) {
	tcs := []testConfig{
		{
			name: "request configuration form",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='get'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='config1'>
			  		<pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
						<configure node='princely_musings'/>
			  		</pubsub>
				</iq>
			`)
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.FetchNodeAffiliationFunc = func(ctx context.Context, jid string, host string, name string) (*pubsubmodel.Affiliation, error) {
					return &pubsubmodel.Affiliation{
						Jid:   "hamlet@denmark.lit",
						State: pubsubmodel.AffiliationState_AFF_OWNER,
					}, nil
				}
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return true, nil
				}
				repMock.FetchNodeFunc = func(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
					return &pubsubmodel.Node{
						Host:    "pubsub.shakespeare.lit",
						Name:    "princely_musings",
						Options: defaultNodeOptions,
					}, nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ResultType, output[0].Type())
				require.Equal(t, "config1", output[0].ID())

				pubsubElement := output[0].ChildNamespace("pubsub", pubSubOwnerNS())
				require.NotNil(t, pubsubElement)

				configureElement := pubsubElement.Child("configure")
				require.NotNil(t, configureElement)

				require.Equal(t, "princely_musings", configureElement.Attribute("node"))

				x := configureElement.ChildNamespace("x", "jabber:x:data")
				require.NotNil(t, x)
			},
		},
		{
			name: "form submission",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
			<iq type='set'
				from='hamlet@denmark.lit/elsinore'
				to='pubsub.shakespeare.lit'
				id='config2'>
			  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
				<configure node='princely_musings'>
				  <x xmlns='jabber:x:data' type='submit'>
					<field var='FORM_TYPE' type='hidden'>
					  <value>http://jabber.org/protocol/pubsub#node_config</value>
					</field>
					<field var='pubsub#title'><value>Princely Musings (Atom)</value></field>
					<field var='pubsub#item_expire'><value>604800</value></field>
					<field var='pubsub#access_model'><value>roster</value></field>
					<field var='pubsub#roster_groups_allowed'>
					  <value>friends</value>
					  <value>servants</value>
					  <value>courtiers</value>
					</field>
					<field var='pubsub#publish_model'><value>publishers</value></field>
					<field var='pubsub#purge_offline'><value>0</value></field>
					<field var='pubsub#max_payload_size'><value>1028</value></field>
					<field var='pubsub#type'><value>urn:example:e2ee:bundle</value></field>
					<field var='pubsub#body_xslt'>
					  <value>http://jabxslt.jabberstudio.org/atom_body.xslt</value>
					</field>
				  </x>
				</configure>
			  </pubsub>
			</iq>
			`)
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.FetchNodeAffiliationFunc = func(ctx context.Context, jid string, host string, name string) (*pubsubmodel.Affiliation, error) {
					return &pubsubmodel.Affiliation{
						Jid:   "hamlet@denmark.lit",
						State: pubsubmodel.AffiliationState_AFF_OWNER,
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
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return true, nil
				}
				repMock.FetchNodeFunc = func(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
					return &pubsubmodel.Node{
						Host:    "pubsub.shakespeare.lit",
						Name:    "princely_musings",
						Options: defaultNodeOptions,
					}, nil
				}
				repMock.UpsertNodeFunc = func(ctx context.Context, node *pubsubmodel.Node) error {
					return nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 2)

				require.Equal(t, stravaganza.MessageName, output[0].Name())
				require.Equal(t, "francisco@denmark.lit", output[0].ToJID().String())

				eventElem := output[0].ChildNamespace("event", pubSubNS("event"))
				require.NotNil(t, eventElem)

				configElem := eventElem.Child("configuration")
				require.NotNil(t, configElem)
				require.Equal(t, "princely_musings", configElem.Attribute("node"))

				x := configElem.ChildNamespace("x", "jabber:x:data")
				require.NotNil(t, x)
				require.Equal(t, "result", x.Attribute("type"))

				require.Equal(t, stravaganza.IQName, output[1].Name())
				require.Equal(t, stravaganza.ResultType, output[1].Type())
				require.Equal(t, "config2", output[1].ID())

				require.Len(t, output[1].AllChildren(), 0)
			},
		},
		{
			name: "configuration process is cancelled",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
			<iq type='set'
				from='hamlet@denmark.lit/elsinore'
				to='pubsub.shakespeare.lit'
				id='config3'>
			  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
				<configure node='princely_musings'>
				  <x xmlns='jabber:x:data' type='cancel'/>
				</configure>
			  </pubsub>
			</iq>
			`)
			},
			setupRepositoryMockFn: func(t *testing.T, repMock *repositoryMock) {
				repMock.FetchNodeAffiliationFunc = func(ctx context.Context, jid string, host string, name string) (*pubsubmodel.Affiliation, error) {
					return &pubsubmodel.Affiliation{
						Jid:   "hamlet@denmark.lit",
						State: pubsubmodel.AffiliationState_AFF_OWNER,
					}, nil
				}
				repMock.NodeExistsFunc = func(ctx context.Context, host, name string) (bool, error) {
					return true, nil
				}
				repMock.FetchNodeFunc = func(ctx context.Context, host, name string) (*pubsubmodel.Node, error) {
					return &pubsubmodel.Node{
						Host:    "pubsub.shakespeare.lit",
						Name:    "princely_musings",
						Options: defaultNodeOptions,
					}, nil
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ResultType, output[0].Type())
				require.Equal(t, "config3", output[0].ID())

				require.Len(t, output[0].AllChildren(), 0)
			},
		},
		{
			name: "failed to configure node due to feature disabled",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
			<iq type='set'
				from='hamlet@denmark.lit/elsinore'
				to='pubsub.shakespeare.lit'
				id='config2'>
			  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
				<configure node='princely_musings'>
				  <x xmlns='jabber:x:data' type='submit'>
					<field var='FORM_TYPE' type='hidden'>
					  <value>http://jabber.org/protocol/pubsub#node_config</value>
					</field>
					<field var='pubsub#title'><value>Princely Musings (Atom)</value></field>
				  </x>
				</configure>
			  </pubsub>
			</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{} // config node feature disabled
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "config2", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("feature-not-implemented"))

				unsupportedElem := errorElem.ChildNamespace("unsupported", errorNS())
				require.NotNil(t, unsupportedElem)
				require.Equal(t, configNode, unsupportedElem.Attribute("feature"))
			},
		},
		{
			name: "failed to configure node due to not authorized",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
			<iq type='set'
				from='hamlet@denmark.lit/elsinore'
				to='pubsub.shakespeare.lit'
				id='config2'>
			  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
				<configure node='princely_musings'>
				  <x xmlns='jabber:x:data' type='submit'>
					<field var='FORM_TYPE' type='hidden'>
					  <value>http://jabber.org/protocol/pubsub#node_config</value>
					</field>
					<field var='pubsub#title'><value>Princely Musings (Atom)</value></field>
				  </x>
				</configure>
			  </pubsub>
			</iq>
				`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					Features: ConfigNode,
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
				require.Equal(t, "config2", output[0].ID())

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
