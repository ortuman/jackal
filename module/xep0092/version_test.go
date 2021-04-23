// Copyright 2020 The jackal Authors
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

package xep0092

import (
	"context"
	"strings"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/version"
	"github.com/stretchr/testify/require"
)

func TestVersion_Features(t *testing.T) {
	// given
	v := &Version{}

	srvFeatures, _ := v.ServerFeatures(context.Background())
	accFeatures, _ := v.AccountFeatures(context.Background())

	// then
	require.Equal(t, []string{versionNamespace}, srvFeatures)
	require.Equal(t, []string(nil), accFeatures)
}

func TestVersion_GetVersion(t *testing.T) {
	// given
	getOSInfo = func(ctx context.Context) string {
		return "Darwin 12.2.0"
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	v := &Version{
		cfg:    Config{ShowOS: true},
		router: routerMock,
	}

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, versionNamespace).
				Build(),
		).
		BuildIQ()

	_ = v.Start(context.Background())
	_ = v.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	resIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))

	query := resIQ.ChildNamespace("query", versionNamespace)
	require.NotNil(t, query)

	name := query.Child("name")
	ver := query.Child("version")
	os := query.Child("os")

	require.NotNil(t, name)
	require.NotNil(t, ver)
	require.NotNil(t, os)

	require.Equal(t, "jackal", name.Text())
	require.Equal(t, strings.TrimPrefix(version.Version.String(), "v"), ver.Text())
	require.Equal(t, "Darwin 12.2.0", os.Text())
}
