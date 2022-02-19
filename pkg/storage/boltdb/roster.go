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

package boltdb

import (
	"context"
	"fmt"
	"sort"

	"github.com/golang/protobuf/proto"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	bolt "go.etcd.io/bbolt"
)

const versionKey = "ver"

type boltDBRosterRep struct {
	tx *bolt.Tx
}

func newRosterRep(tx *bolt.Tx) *boltDBRosterRep {
	return &boltDBRosterRep{tx: tx}
}

func (r *boltDBRosterRep) TouchRosterVersion(_ context.Context, username string) (int, error) {
	var ver *rostermodel.Version

	fetchOp := fetchKeyOp{
		tx:     r.tx,
		bucket: rosterVersionBucketKey(username),
		key:    versionKey,
		obj:    &rostermodel.Version{},
	}
	obj, err := fetchOp.do()
	if err != nil {
		return 0, err
	}
	switch {
	case obj != nil:
		ver = obj.(*rostermodel.Version)
		ver.Version++
	default:
		ver = &rostermodel.Version{Version: 1}
	}

	upsertOp := upsertKeyOp{
		tx:     r.tx,
		bucket: rosterVersionBucketKey(username),
		key:    versionKey,
		obj:    ver,
	}
	if err := upsertOp.do(); err != nil {
		return 0, err
	}
	return int(ver.Version), nil
}

func (r *boltDBRosterRep) FetchRosterVersion(_ context.Context, username string) (int, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: rosterVersionBucketKey(username),
		key:    versionKey,
		obj:    &rostermodel.Version{},
	}
	obj, err := op.do()
	if err != nil {
		return 0, err
	}
	switch {
	case obj != nil:
		return int(obj.(*rostermodel.Version).Version), nil
	default:
		return 0, nil
	}
}

func (r *boltDBRosterRep) UpsertRosterItem(_ context.Context, ri *rostermodel.Item) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: rosterItemsBucketKey(ri.Username),
		key:    ri.Jid,
		obj:    ri,
	}
	return op.do()
}

func (r *boltDBRosterRep) DeleteRosterItem(_ context.Context, username, jid string) error {
	op := delKeyOp{
		tx:     r.tx,
		bucket: rosterItemsBucketKey(username),
		key:    jid,
	}
	return op.do()
}

func (r *boltDBRosterRep) DeleteRosterItems(_ context.Context, username string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: rosterItemsBucketKey(username),
	}
	return op.do()
}

func (r *boltDBRosterRep) FetchRosterItems(_ context.Context, username string) ([]*rostermodel.Item, error) {
	var retVal []*rostermodel.Item

	op := iterKeysOp{
		tx:     r.tx,
		bucket: rosterItemsBucketKey(username),
		iterFn: func(_, b []byte) error {
			var itm rostermodel.Item
			if err := proto.Unmarshal(b, &itm); err != nil {
				return err
			}
			retVal = append(retVal, &itm)
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func (r *boltDBRosterRep) FetchRosterItemsInGroups(_ context.Context, username string, groups []string) ([]*rostermodel.Item, error) {
	var retVal []*rostermodel.Item

	groupsMap := make(map[string]struct{}, len(groups))
	for _, gr := range groups {
		groupsMap[gr] = struct{}{}
	}
	op := iterKeysOp{
		tx:     r.tx,
		bucket: rosterItemsBucketKey(username),
		iterFn: func(_, b []byte) error {
			var itm rostermodel.Item
			if err := proto.Unmarshal(b, &itm); err != nil {
				return err
			}
			for _, gr := range itm.Groups {
				_, ok := groupsMap[gr]
				if ok {
					// item in group
					retVal = append(retVal, &itm)
					return nil
				}
			}
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func (r *boltDBRosterRep) FetchRosterItem(_ context.Context, username, jid string) (*rostermodel.Item, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: rosterItemsBucketKey(username),
		key:    jid,
		obj:    &rostermodel.Item{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*rostermodel.Item), nil
	default:
		return nil, nil
	}
}

func (r *boltDBRosterRep) FetchRosterGroups(_ context.Context, username string) ([]string, error) {
	groupsMap := make(map[string]struct{})

	op := iterKeysOp{
		tx:     r.tx,
		bucket: rosterItemsBucketKey(username),
		iterFn: func(_, b []byte) error {
			var itm rostermodel.Item
			if err := proto.Unmarshal(b, &itm); err != nil {
				return err
			}
			for _, gr := range itm.Groups {
				groupsMap[gr] = struct{}{}
			}
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	var retVal []string

	for gr := range groupsMap {
		retVal = append(retVal, gr)
	}
	sort.Slice(retVal, func(i, j int) bool { return retVal[i] < retVal[j] })

	return retVal, nil
}

func (r *boltDBRosterRep) UpsertRosterNotification(_ context.Context, rn *rostermodel.Notification) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: rosterNotificationsBucketKey(rn.Contact),
		key:    rn.Jid,
		obj:    rn,
	}
	return op.do()
}

func (r *boltDBRosterRep) DeleteRosterNotification(_ context.Context, contact, jid string) error {
	op := delKeyOp{
		tx:     r.tx,
		bucket: rosterNotificationsBucketKey(contact),
		key:    jid,
	}
	return op.do()
}

func (r *boltDBRosterRep) DeleteRosterNotifications(_ context.Context, contact string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: rosterNotificationsBucketKey(contact),
	}
	return op.do()
}

func (r *boltDBRosterRep) FetchRosterNotification(_ context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: rosterNotificationsBucketKey(contact),
		key:    jid,
		obj:    &rostermodel.Notification{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*rostermodel.Notification), nil
	default:
		return nil, nil
	}
}

func (r *boltDBRosterRep) FetchRosterNotifications(_ context.Context, contact string) ([]*rostermodel.Notification, error) {
	var retVal []*rostermodel.Notification

	op := iterKeysOp{
		tx:     r.tx,
		bucket: rosterNotificationsBucketKey(contact),
		iterFn: func(_, b []byte) error {
			var not rostermodel.Notification
			if err := proto.Unmarshal(b, &not); err != nil {
				return err
			}
			retVal = append(retVal, &not)
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func rosterVersionBucketKey(username string) string {
	return fmt.Sprintf("roster:ver:%s", username)
}

func rosterItemsBucketKey(username string) string {
	return fmt.Sprintf("roster:items:%s", username)
}

func rosterNotificationsBucketKey(username string) string {
	return fmt.Sprintf("roster:notif:%s", username)
}

// TouchRosterVersion satisfies repository.Roster interface.
func (r *Repository) TouchRosterVersion(ctx context.Context, username string) (v int, err error) {
	err = r.db.Update(func(tx *bolt.Tx) error {
		v, err = newRosterRep(tx).TouchRosterVersion(ctx, username)
		return err
	})
	return
}

// FetchRosterVersion satisfies repository.Roster interface.
func (r *Repository) FetchRosterVersion(ctx context.Context, username string) (v int, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		v, err = newRosterRep(tx).FetchRosterVersion(ctx, username)
		return err
	})
	return
}

// UpsertRosterItem satisfies repository.Roster interface.
func (r *Repository) UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newRosterRep(tx).UpsertRosterItem(ctx, ri)
	})
}

// DeleteRosterItem satisfies repository.Roster interface.
func (r *Repository) DeleteRosterItem(ctx context.Context, username, jid string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newRosterRep(tx).DeleteRosterItem(ctx, username, jid)
	})
}

// DeleteRosterItems satisfies repository.Roster interface.
func (r *Repository) DeleteRosterItems(ctx context.Context, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newRosterRep(tx).DeleteRosterItems(ctx, username)
	})
}

// FetchRosterItems satisfies repository.Roster interface.
func (r *Repository) FetchRosterItems(ctx context.Context, username string) (items []*rostermodel.Item, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		items, err = newRosterRep(tx).FetchRosterItems(ctx, username)
		return err
	})
	return
}

// FetchRosterItemsInGroups satisfies repository.Roster interface.
func (r *Repository) FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) (items []*rostermodel.Item, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		items, err = newRosterRep(tx).FetchRosterItemsInGroups(ctx, username, groups)
		return err
	})
	return
}

// FetchRosterItem satisfies repository.Roster interface.
func (r *Repository) FetchRosterItem(ctx context.Context, username, jid string) (item *rostermodel.Item, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		item, err = newRosterRep(tx).FetchRosterItem(ctx, username, jid)
		return err
	})
	return
}

// UpsertRosterNotification satisfies repository.Roster interface.
func (r *Repository) UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newRosterRep(tx).UpsertRosterNotification(ctx, rn)
	})
}

// DeleteRosterNotification satisfies repository.Roster interface.
func (r *Repository) DeleteRosterNotification(ctx context.Context, contact, jid string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newRosterRep(tx).DeleteRosterNotification(ctx, contact, jid)
	})
}

// DeleteRosterNotifications satisfies repository.Roster interface.
func (r *Repository) DeleteRosterNotifications(ctx context.Context, contact string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newRosterRep(tx).DeleteRosterNotifications(ctx, contact)
	})
}

// FetchRosterNotification satisfies repository.Roster interface.
func (r *Repository) FetchRosterNotification(ctx context.Context, contact string, jid string) (n *rostermodel.Notification, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		n, err = newRosterRep(tx).FetchRosterNotification(ctx, contact, jid)
		return err
	})
	return
}

// FetchRosterNotifications satisfies repository.Roster interface.
func (r *Repository) FetchRosterNotifications(ctx context.Context, contact string) (ns []*rostermodel.Notification, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		ns, err = newRosterRep(tx).FetchRosterNotifications(ctx, contact)
		return err
	})
	return
}

// FetchRosterGroups satisfies repository.Roster interface.
func (r *Repository) FetchRosterGroups(ctx context.Context, username string) (groups []string, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		groups, err = newRosterRep(tx).FetchRosterGroups(ctx, username)
		return err
	})
	return
}
