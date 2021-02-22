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

	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/stretchr/testify/require"
)

func TestMeasuredRosterRep_TouchRosterVersion(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 1, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.TouchRosterVersion(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.TouchRosterVersionCalls(), 1)
}

func TestMeasuredRosterRep_FetchRosterVersion(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 1, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.FetchRosterVersion(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchRosterVersionCalls(), 1)
}

func TestMeasuredRosterRep_UpsertRosterItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertRosterItemFunc = func(ctx context.Context, ri *rostermodel.Item) error {
		return nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_ = m.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username: "ortuman", JID: "noelia@jackal.im",
	})

	// then
	require.Len(t, repMock.UpsertRosterItemCalls(), 1)
}

func TestMeasuredRosterRep_DeleteRosterItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteRosterItemFunc = func(ctx context.Context, username string, jid string) error {
		return nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_ = m.DeleteRosterItem(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Len(t, repMock.DeleteRosterItemCalls(), 1)
}

func TestMeasuredRosterRep_FetchRosterItems(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]rostermodel.Item, error) {
		return nil, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.FetchRosterItems(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchRosterItemsCalls(), 1)
}

func TestMeasuredRosterRep_FetchRosterItemsInGroups(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterItemsInGroupsFunc = func(ctx context.Context, username string, groups []string) ([]rostermodel.Item, error) {
		return nil, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.FetchRosterItemsInGroups(context.Background(), "ortuman", []string{"buddies"})

	// then
	require.Len(t, repMock.FetchRosterItemsInGroupsCalls(), 1)
}

func TestMeasuredRosterRep_FetchRosterItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		return nil, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Len(t, repMock.FetchRosterItemCalls(), 1)
}

func TestMeasuredRosterRep_UpsertRosterNotification(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertRosterNotificationFunc = func(ctx context.Context, rn *rostermodel.Notification) error {
		return nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_ = m.UpsertRosterNotification(context.Background(), &rostermodel.Notification{})

	// then
	require.Len(t, repMock.UpsertRosterNotificationCalls(), 1)
}

func TestMeasuredRosterRep_DeleteRosterNotification(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteRosterNotificationFunc = func(ctx context.Context, contact string, jid string) error {
		return nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_ = m.DeleteRosterNotification(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Len(t, repMock.DeleteRosterNotificationCalls(), 1)
}

func TestMeasuredRosterRep_FetchRosterNotification(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterNotificationFunc = func(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
		return nil, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.FetchRosterNotification(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Len(t, repMock.FetchRosterNotificationCalls(), 1)
}

func TestMeasuredRosterRep_FetchRosterNotifications(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterNotificationsFunc = func(ctx context.Context, contact string) ([]rostermodel.Notification, error) {
		return nil, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.FetchRosterNotifications(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchRosterNotificationsCalls(), 1)
}

func TestMeasuredRosterRep_FetchRosterGroups(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterGroupsFunc = func(ctx context.Context, username string) ([]string, error) {
		return nil, nil
	}
	m := &measuredRosterRep{rep: repMock}

	// when
	_, _ = m.FetchRosterGroups(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchRosterGroupsCalls(), 1)
}
