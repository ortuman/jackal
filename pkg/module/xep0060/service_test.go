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
	"math"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	xmppparser "github.com/jackal-xmpp/stravaganza/parser"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/stretchr/testify/require"
)

var defaultNodeOptions = &pubsubmodel.Options{
	AccessModel:          pubsubmodel.NodeAccessModel2String[pubsubmodel.NodeAccessModel_NAM_OPEN],
	MaxPayloadSize:       64 * 1024,
	MaxItems:             120,
	DeliverNotifications: true,
	DeliverPayloads:      true,
	NotificationType:     stravaganza.NormalType,
	NotifyConfig:         true,
	NotifyDelete:         true,
	NotifySub:            true,
}

var testDefaultServiceConfig = ServiceConfig{
	DefaultNodeOptions: defaultNodeOptions,
	Features:           ServiceFeatures(CreateNodes | ConfigNode | InstantNodes | RetrieveItems | RetrieveSubscriptions | RetrieveAffiliations),
}

type testConfig struct {
	name                  string
	serviceConfigFn       func() ServiceConfig
	inputFn               func() *stravaganza.IQ
	setupRepositoryMockFn func(t *testing.T, repMock *repositoryMock)
	assertOutputFn        func(t *testing.T, output []stravaganza.Stanza)
	expectedError         error
}

func runServiceTest(t *testing.T, tCfg testConfig) {
	serviceConfig := testDefaultServiceConfig
	if tCfg.serviceConfigFn != nil {
		serviceConfig = tCfg.serviceConfigFn()
	}
	svc, routerMock, repMock := testService(serviceConfig)
	if tCfg.setupRepositoryMockFn != nil {
		tCfg.setupRepositoryMockFn(t, repMock)
	}

	var output []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		output = append(output, stanza)
		return nil, nil
	}

	err := svc.ProcessIQ(context.Background(), tCfg.inputFn())
	require.Equal(t, tCfg.expectedError, err)

	if tCfg.assertOutputFn != nil {
		tCfg.assertOutputFn(t, output)
	}
}

func testService(cfg ServiceConfig) (*Service, *routerMock, *repositoryMock) {
	routerMock := &routerMock{}
	repMock := &repositoryMock{}

	return NewService(
		cfg,
		routerMock,
		repMock,
		log.NewNopLogger(),
	), routerMock, repMock
}

func parseIQ(t *testing.T, s string) *stravaganza.IQ {
	t.Helper()

	p := xmppparser.New(strings.NewReader(s), xmppparser.DefaultMode, math.MaxInt)
	elem, err := p.Parse()
	require.NoError(t, err)

	iq, err := stravaganza.NewBuilderFromElement(elem).
		BuildIQ()
	require.NoError(t, err)
	return iq
}
