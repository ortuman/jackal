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

package measuredrepository

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/stretchr/testify/require"
)

func TestMeasuredVCardRep_UpsertVCard(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertVCardFunc = func(ctx context.Context, vCard stravaganza.Element, username string) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.UpsertVCard(context.Background(), stravaganza.NewBuilder("vCard").Build(), "ortuman")

	// then
	require.Len(t, repMock.UpsertVCardCalls(), 1)
}

func TestMeasuredVCardRep_FetchVCard(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchVCardFunc = func(ctx context.Context, username string) (stravaganza.Element, error) {
		return nil, nil
	}
	m := New(repMock)

	// when
	_, _ = m.FetchVCard(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchVCardCalls(), 1)
}

func TestMeasuredVCardRep_DeleteVCard(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteVCardFunc = func(ctx context.Context, username string) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.DeleteVCard(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.DeleteVCardCalls(), 1)
}
