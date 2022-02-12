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

package cachedrepository

import (
	"context"
	"testing"

	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/stretchr/testify/require"
)

func TestCachedRosterRep_TouchVersion(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 5, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	ver, err := rep.TouchRosterVersion(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, rosterVersionKey, cacheKey)
	require.Equal(t, 5, ver)
	require.Len(t, repMock.TouchRosterVersionCalls(), 1)
}

func TestCachedUserRep_FetchRosterVersion(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 5, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	ver, err := rep.FetchRosterVersion(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, 5, ver)

	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, rosterVersionKey, cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterVersionCalls(), 1)
}

func TestCachedRosterRep_UpsertRosterItem(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertRosterItemFunc = func(ctx context.Context, ri *rostermodel.Item) error {
		return nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username: "u1",
		Jid:      "foo@jackal.im",
	})

	// then
	require.NoError(t, err)
	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Len(t, repMock.UpsertRosterItemCalls(), 1)
}

func TestCachedRosterRep_DeleteRosterItem(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteRosterItemFunc = func(ctx context.Context, username string, jid string) error {
		return nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteRosterItem(context.Background(), "u1", "foo@jackal.im")

	// then
	require.NoError(t, err)
	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Len(t, repMock.DeleteRosterItemCalls(), 1)
}

func TestCachedRosterRep_DeleteRosterItems(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteRosterItemsFunc = func(ctx context.Context, username string) error {
		return nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteRosterItems(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Len(t, repMock.DeleteRosterItemsCalls(), 1)
}

func TestCachedRosterRep_FetchRosterItems(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]*rostermodel.Item, error) {
		return []*rostermodel.Item{
			{Username: "u1", Jid: "foo@jackal.im"},
		}, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	items, err := rep.FetchRosterItems(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "u1", items[0].Username)

	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, rosterItemsKey, cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterItemsCalls(), 1)
}

func TestCachedRosterRep_FetchRosterItemsInGroups(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterItemsInGroupsFunc = func(ctx context.Context, username string, groups []string) ([]*rostermodel.Item, error) {
		return []*rostermodel.Item{
			{Username: "u1", Jid: "foo@jackal.im"},
		}, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	items, err := rep.FetchRosterItemsInGroups(context.Background(), "u1", []string{"g1"})

	// then
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "u1", items[0].Username)

	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, rosterGroupsSliceKey([]string{"g1"}), cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterItemsInGroupsCalls(), 1)
}

func TestCachedRosterRep_FetchRosterItem(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		return &rostermodel.Item{Username: "u1", Jid: "foo@jackal.im"}, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	itm, err := rep.FetchRosterItem(context.Background(), "u1", "foo@jackal.im")

	// then
	require.NoError(t, err)
	require.NotNil(t, itm)
	require.Equal(t, "u1", itm.Username)

	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, "foo@jackal.im", cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterItemCalls(), 1)
}

func TestCachedRosterRep_UpsertRosterNotification(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertRosterNotificationFunc = func(ctx context.Context, rn *rostermodel.Notification) error {
		return nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertRosterNotification(context.Background(), &rostermodel.Notification{
		Contact: "c1",
		Jid:     "foo@jackal.im",
	})

	// then
	require.NoError(t, err)
	require.Equal(t, rosterNotificationsNS("c1"), cacheNS)
	require.Len(t, repMock.UpsertRosterNotificationCalls(), 1)
}

func TestCachedRosterRep_DeleteRosterNotification(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteRosterNotificationFunc = func(ctx context.Context, contact string, jid string) error {
		return nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteRosterNotification(context.Background(), "c1", "foo@jackal.im")

	// then
	require.NoError(t, err)
	require.Equal(t, rosterNotificationsNS("c1"), cacheNS)
	require.Len(t, repMock.DeleteRosterNotificationCalls(), 1)
}

func TestCachedRosterRep_DeleteRosterNotifications(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteRosterNotificationsFunc = func(ctx context.Context, contact string) error {
		return nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteRosterNotifications(context.Background(), "c1")

	// then
	require.NoError(t, err)
	require.Equal(t, rosterNotificationsNS("c1"), cacheNS)
	require.Len(t, repMock.DeleteRosterNotificationsCalls(), 1)
}

func TestCachedRosterRep_FetchRosterNotification(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterNotificationFunc = func(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
		return &rostermodel.Notification{Contact: "c1", Jid: "foo@jackal.im"}, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	not, err := rep.FetchRosterNotification(context.Background(), "c1", "foo@jackal.im")

	// then
	require.NoError(t, err)
	require.NotNil(t, not)
	require.Equal(t, "c1", not.Contact)

	require.Equal(t, rosterNotificationsNS("c1"), cacheNS)
	require.Equal(t, "foo@jackal.im", cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterNotificationCalls(), 1)
}

func TestCachedRosterRep_FetchRosterNotifications(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterNotificationsFunc = func(ctx context.Context, contact string) ([]*rostermodel.Notification, error) {
		return []*rostermodel.Notification{
			{Contact: "c1", Jid: "foo@jackal.im"},
		}, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	ns, err := rep.FetchRosterNotifications(context.Background(), "c1")

	// then
	require.NoError(t, err)
	require.Len(t, ns, 1)
	require.Equal(t, "c1", ns[0].Contact)

	require.Equal(t, rosterNotificationsNS("c1"), cacheNS)
	require.Equal(t, rosterNotificationsKey, cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterNotificationsCalls(), 1)
}

func TestCachedRosterRep_FetchRosterGroups(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterGroupsFunc = func(ctx context.Context, username string) ([]string, error) {
		return []string{"buddies"}, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	groups, err := rep.FetchRosterGroups(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, []string{"buddies"}, groups)

	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, rosterGroupsKey, cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterGroupsCalls(), 1)
}
