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
	"fmt"
	"sort"
	"strings"

	"github.com/go-kit/log"
	"github.com/ortuman/jackal/pkg/model"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const (
	rosterVersionKey       = "ver"
	rosterItemsKey         = "items"
	rosterNotificationsKey = "notifications"
	rosterGroupsKey        = "groups"
)

type cachedRosterRep struct {
	c      Cache
	rep    repository.Roster
	logger log.Logger
}

func (c *cachedRosterRep) TouchRosterVersion(ctx context.Context, username string) (int, error) {
	var ver int
	var err error

	op := updateOp{
		c:              c.c,
		namespace:      rosterItemsNS(username),
		invalidateKeys: []string{rosterVersionKey},
		updateFn: func(ctx context.Context) error {
			ver, err = c.rep.TouchRosterVersion(ctx, username)
			return err
		},
	}
	if err := op.do(ctx); err != nil {
		return 0, err
	}
	return ver, nil
}

func (c *cachedRosterRep) FetchRosterVersion(ctx context.Context, username string) (int, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       rosterVersionKey,
		codec:     &rostermodel.Version{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			ver, err := c.rep.FetchRosterVersion(ctx, username)
			if err != nil {
				return nil, err
			}
			return &rostermodel.Version{Version: int32(ver)}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return 0, err
	case v != nil:
		return int(v.(*rostermodel.Version).Version), nil
	}
	return 0, nil
}

func (c *cachedRosterRep) UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) error {
	op := updateOp{
		c:         c.c,
		namespace: rosterItemsNS(ri.Username),
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertRosterItem(ctx, ri)
		},
	}
	return op.do(ctx)
}

func (c *cachedRosterRep) DeleteRosterItem(ctx context.Context, username, jid string) error {
	op := updateOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteRosterItem(ctx, username, jid)
		},
	}
	return op.do(ctx)
}

func (c *cachedRosterRep) DeleteRosterItems(ctx context.Context, username string) error {
	op := updateOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteRosterItems(ctx, username)
		},
	}
	return op.do(ctx)
}

func (c *cachedRosterRep) FetchRosterItems(ctx context.Context, username string) ([]*rostermodel.Item, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       rosterItemsKey,
		codec:     &rostermodel.Items{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			items, err := c.rep.FetchRosterItems(ctx, username)
			if err != nil {
				return nil, err
			}
			return &rostermodel.Items{Items: items}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*rostermodel.Items).Items, nil
	}
	return nil, nil
}

func (c *cachedRosterRep) FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]*rostermodel.Item, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       rosterGroupsSliceKey(groups),
		codec:     &rostermodel.Items{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			items, err := c.rep.FetchRosterItemsInGroups(ctx, username, groups)
			if err != nil {
				return nil, err
			}
			return &rostermodel.Items{Items: items}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*rostermodel.Items).Items, nil
	}
	return nil, nil
}

func (c *cachedRosterRep) FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       jid,
		codec:     &rostermodel.Item{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return c.rep.FetchRosterItem(ctx, username, jid)
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*rostermodel.Item), nil
	}
	return nil, nil
}

func (c *cachedRosterRep) UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error {
	op := updateOp{
		c:         c.c,
		namespace: rosterNotificationsNS(rn.Contact),
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertRosterNotification(ctx, rn)
		},
	}
	return op.do(ctx)
}

func (c *cachedRosterRep) DeleteRosterNotification(ctx context.Context, contact, jid string) error {
	op := updateOp{
		c:         c.c,
		namespace: rosterNotificationsNS(contact),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteRosterNotification(ctx, contact, jid)
		},
	}
	return op.do(ctx)
}

func (c *cachedRosterRep) DeleteRosterNotifications(ctx context.Context, contact string) error {
	op := updateOp{
		c:         c.c,
		namespace: rosterNotificationsNS(contact),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteRosterNotifications(ctx, contact)
		},
	}
	return op.do(ctx)
}

func (c *cachedRosterRep) FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterNotificationsNS(contact),
		key:       jid,
		codec:     &rostermodel.Notification{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return c.rep.FetchRosterNotification(ctx, contact, jid)
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*rostermodel.Notification), nil
	}
	return nil, nil
}

func (c *cachedRosterRep) FetchRosterNotifications(ctx context.Context, contact string) ([]*rostermodel.Notification, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterNotificationsNS(contact),
		key:       rosterNotificationsKey,
		codec:     &rostermodel.Notifications{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			ns, err := c.rep.FetchRosterNotifications(ctx, contact)
			if err != nil {
				return nil, err
			}
			return &rostermodel.Notifications{Notifications: ns}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*rostermodel.Notifications).Notifications, nil
	}
	return nil, nil
}

func (c *cachedRosterRep) FetchRosterGroups(ctx context.Context, username string) ([]string, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       rosterGroupsKey,
		codec:     &rostermodel.Groups{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			grs, err := c.rep.FetchRosterGroups(ctx, username)
			if err != nil {
				return nil, err
			}
			return &rostermodel.Groups{Groups: grs}, nil
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*rostermodel.Groups).Groups, nil
	}
	return nil, nil
}

func rosterItemsNS(username string) string {
	return fmt.Sprintf("ros:items:%s", username)
}

func rosterNotificationsNS(contact string) string {
	return fmt.Sprintf("ros:notif:%s", contact)
}

func rosterGroupsSliceKey(groups []string) string {
	sortedGroups := make([]string, len(groups))
	copy(sortedGroups, groups)

	sort.Slice(sortedGroups, func(i, j int) bool {
		return sortedGroups[i] < sortedGroups[j]
	})
	return fmt.Sprintf("groups:%s", strings.Join(sortedGroups, "|"))
}
