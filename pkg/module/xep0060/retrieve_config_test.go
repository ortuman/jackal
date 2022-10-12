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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jackal-xmpp/stravaganza"
)

func TestService_Default(t *testing.T) {
	tcs := []testConfig{
		{
			name: "request default subscription configuration options",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='get'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='def1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<default/>
				  </pubsub>
				</iq>
			`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					DefaultNodeOptions:     defaultNodeOptions,
					Features:               ConfigNode,
					ConfigRetrievalEnabled: true,
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ResultType, output[0].Type())
				require.Equal(t, "def1", output[0].ID())

				pubSubElem := output[0].ChildNamespace("pubsub", pubSubOwnerNS())
				require.NotNil(t, pubSubElem)

				defaultElement := pubSubElem.Child("default")
				require.NotNil(t, defaultElement)

				x := defaultElement.ChildNamespace("x", "jabber:x:data")
				require.NotNil(t, x)
			},
		},
		{
			name: "service does not support node configuration",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='get'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='def1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<default/>
				  </pubsub>
				</iq>
			`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{} // node configuration disabled
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "def1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("feature-not-implemented"))

				unsupportedElem := errorElem.ChildNamespace("unsupported", errorNS())
				require.NotNil(t, unsupportedElem)
				require.Equal(t, configNode, unsupportedElem.Attribute("feature"))
			},
		},
		{
			name: "service does not support retrieval of default node configuration",
			inputFn: func() *stravaganza.IQ {
				return parseIQ(t, `
				<iq type='get'
					from='hamlet@denmark.lit/elsinore'
					to='pubsub.shakespeare.lit'
					id='def1'>
				  <pubsub xmlns='http://jabber.org/protocol/pubsub#owner'>
					<default/>
				  </pubsub>
				</iq>
			`)
			},
			serviceConfigFn: func() ServiceConfig {
				return ServiceConfig{
					DefaultNodeOptions:     defaultNodeOptions,
					Features:               ConfigNode,
					ConfigRetrievalEnabled: false,
				}
			},
			assertOutputFn: func(t *testing.T, output []stravaganza.Stanza) {
				require.Len(t, output, 1)

				require.Equal(t, stravaganza.IQName, output[0].Name())
				require.Equal(t, stravaganza.ErrorType, output[0].Type())
				require.Equal(t, "def1", output[0].ID())

				errorElem := output[0].Child("error")
				require.NotNil(t, errorElem)

				require.NotNil(t, errorElem.Child("feature-not-implemented"))

				unsupportedElem := errorElem.ChildNamespace("unsupported", errorNS())
				require.NotNil(t, unsupportedElem)
				require.Equal(t, retrieveDefault, unsupportedElem.Attribute("feature"))
			},
		},
	}
	for _, tCfg := range tcs {
		t.Run(tCfg.name, func(t *testing.T) {
			runServiceTest(t, tCfg)
		})
	}
}
