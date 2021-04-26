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

package measuredrepository

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/stretchr/testify/require"
)

func TestMeasuredOfflineRep_InsertOfflineMessage(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.InsertOfflineMessageFunc = func(ctx context.Context, message *stravaganza.Message, username string) error {
		return nil
	}
	m := &measuredOfflineRep{rep: repMock}

	// when
	_ = m.InsertOfflineMessage(context.Background(), nil, "ortuman")

	// then
	require.Len(t, repMock.InsertOfflineMessageCalls(), 1)
}

func TestMeasuredOfflineRep_CountOfflineMessage(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.CountOfflineMessagesFunc = func(ctx context.Context, username string) (int, error) {
		return 1, nil
	}
	m := &measuredOfflineRep{rep: repMock}

	// when
	c, _ := m.CountOfflineMessages(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.CountOfflineMessagesCalls(), 1)
	require.Equal(t, 1, c)
}

func TestMeasuredOfflineRep_FetchOfflineMessage(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchOfflineMessagesFunc = func(ctx context.Context, username string) ([]*stravaganza.Message, error) {
		return nil, nil
	}
	m := &measuredOfflineRep{rep: repMock}

	// when
	_, _ = m.FetchOfflineMessages(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchOfflineMessagesCalls(), 1)
}

func TestMeasuredOfflineRep_DeleteOfflineMessage(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteOfflineMessagesFunc = func(ctx context.Context, username string) error {
		return nil
	}
	m := &measuredOfflineRep{rep: repMock}

	// when
	_ = m.DeleteOfflineMessages(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.DeleteOfflineMessagesCalls(), 1)
}
