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
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const (
	rosterVersionKey       = "ver"
	rosterItemsKey         = "items"
	rosterNotificationsKey = "notifications"
	rosterGroupsKey        = "groups"
)

type rosterVersionCodec struct {
	val *types.Int64Value
}

func (c *rosterVersionCodec) encode(i interface{}) ([]byte, error) {
	val := &types.Int64Value{
		Value: int64(i.(int)),
	}
	return proto.Marshal(val)
}

func (c *rosterVersionCodec) decode(b []byte) error {
	var v types.Int64Value
	if err := proto.Unmarshal(b, &v); err != nil {
		return err
	}
	c.val = &v
	return nil
}

func (c *rosterVersionCodec) value() interface{} {
	return int(c.val.Value)
}

type rosterGroupsCodec struct {
	val *rostermodel.Groups
}

func (c *rosterGroupsCodec) encode(i interface{}) ([]byte, error) {
	val := &rostermodel.Groups{
		Groups: i.([]string),
	}
	return proto.Marshal(val)
}

func (c *rosterGroupsCodec) decode(b []byte) error {
	var gr rostermodel.Groups
	if err := proto.Unmarshal(b, &gr); err != nil {
		return err
	}
	c.val = &gr
	return nil
}

func (c *rosterGroupsCodec) value() interface{} {
	return c.val.Groups
}

type rosterItemCodec struct {
	val *rostermodel.Item
}

func (c *rosterItemCodec) encode(i interface{}) ([]byte, error) {
	return proto.Marshal(i.(*rostermodel.Item))
}

func (c *rosterItemCodec) decode(b []byte) error {
	var itm rostermodel.Item
	if err := proto.Unmarshal(b, &itm); err != nil {
		return err
	}
	c.val = &itm
	return nil
}

func (c *rosterItemCodec) value() interface{} {
	return c.val.Groups
}

type rosterItemsCodec struct {
	val *rostermodel.Items
}

func (c *rosterItemsCodec) encode(i interface{}) ([]byte, error) {
	items := rostermodel.Items{
		Items: i.([]*rostermodel.Item),
	}
	return proto.Marshal(&items)
}

func (c *rosterItemsCodec) decode(b []byte) error {
	var items rostermodel.Items
	if err := proto.Unmarshal(b, &items); err != nil {
		return err
	}
	c.val = &items
	return nil
}

func (c *rosterItemsCodec) value() interface{} {
	return c.val.Items
}

type rosterNotificationCodec struct {
	val *rostermodel.Notification
}

func (c *rosterNotificationCodec) encode(i interface{}) ([]byte, error) {
	return proto.Marshal(i.(*rostermodel.Notification))
}

func (c *rosterNotificationCodec) decode(b []byte) error {
	var n rostermodel.Notification
	if err := proto.Unmarshal(b, &n); err != nil {
		return err
	}
	c.val = &n
	return nil
}

func (c *rosterNotificationCodec) value() interface{} {
	return c.val
}

type rosterNotificationsCodec struct {
	val *rostermodel.Notifications
}

func (c *rosterNotificationsCodec) encode(i interface{}) ([]byte, error) {
	ns := rostermodel.Notifications{
		Notifications: i.([]*rostermodel.Notification),
	}
	return proto.Marshal(&ns)
}

func (c *rosterNotificationsCodec) decode(b []byte) error {
	var ns rostermodel.Notifications
	if err := proto.Unmarshal(b, &ns); err != nil {
		return err
	}
	c.val = &ns
	return nil
}

func (c *rosterNotificationsCodec) value() interface{} {
	return c.val.Notifications
}

type cachedRosterRep struct {
	c   Cache
	rep repository.Roster
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
		codec:     &rosterVersionCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchRosterVersion(ctx, username)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return 0, err
	case v != nil:
		return v.(int), nil
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
		codec:     &rosterItemsCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchRosterItems(ctx, username)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.([]*rostermodel.Item), nil
	}
	return nil, nil
}

func (c *cachedRosterRep) FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]*rostermodel.Item, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       rosterGroupsSliceKey(groups),
		codec:     &rosterItemsCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchRosterItemsInGroups(ctx, username, groups)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.([]*rostermodel.Item), nil
	}
	return nil, nil
}

func (c *cachedRosterRep) FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       jid,
		codec:     &rosterItemCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchRosterItem(ctx, username, jid)
		},
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
		codec:     &rosterNotificationCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchRosterNotification(ctx, contact, jid)
		},
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
		codec:     &rosterNotificationsCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchRosterNotifications(ctx, contact)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.([]*rostermodel.Notification), nil
	}
	return nil, nil
}

func (c *cachedRosterRep) FetchRosterGroups(ctx context.Context, username string) ([]string, error) {
	op := fetchOp{
		c:         c.c,
		namespace: rosterItemsNS(username),
		key:       rosterGroupsKey,
		codec:     &rosterGroupsCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchRosterGroups(ctx, username)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.([]string), nil
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
	return fmt.Sprintf("groups:%s", strings.Join(groups, "|"))
}
