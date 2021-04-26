// Copyright 2021 The jackal Authors
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

package xep0202

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/stretchr/testify/require"
)

func TestTime_GetTime(t *testing.T) {
	// given
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	m := &Time{
		router: routerMock,
		tmFn: func() time.Time {
			return time.Date(1984, 01, 03, 00, 00, 00, 00, time.UTC)
		},
	}

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, uuid.New().String()).
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "jackal.im").
		WithChild(
			stravaganza.NewBuilder("time").
				WithAttribute(stravaganza.Namespace, timeNamespace).
				Build(),
		).
		BuildIQ()
	_ = m.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	require.Equal(t, "iq", respStanzas[0].Name())
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Attribute(stravaganza.Type))

	tm := respStanzas[0].ChildNamespace("time", timeNamespace)
	require.NotNil(t, tm)

	tzo := tm.Child("tzo")
	utc := tm.Child("utc")

	require.Equal(t, "+00:00", tzo.Text())
	require.Equal(t, "1984-01-03T00:00:00Z", utc.Text())
}
