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

	"github.com/golang/protobuf/proto"
	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	bolt "go.etcd.io/bbolt"
)

type boltDBPubSubRep struct {
	tx *bolt.Tx
}

func newPubSubRep(tx *bolt.Tx) *boltDBPubSubRep {
	return &boltDBPubSubRep{tx: tx}
}

func (r *boltDBPubSubRep) UpsertNode(_ context.Context, node *pubsubmodel.Node) error {
	bucket := pubSubNodeBucketKey(node.Host)

	b, err := r.tx.CreateBucketIfNotExists([]byte(bucket))
	if err != nil {
		return err
	}
	id, err := b.NextSequence()
	if err != nil {
		return err
	}
	node.Id = int64(id)

	p, err := node.MarshalBinary()
	if err != nil {
		return err
	}
	return b.Put([]byte(node.Name), p)
}

func (r *boltDBPubSubRep) FetchNode(_ context.Context, host, name string) (*pubsubmodel.Node, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: pubSubNodeBucketKey(host),
		key:    name,
		obj:    &pubsubmodel.Node{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*pubsubmodel.Node), nil
	default:
		return nil, nil
	}
}

func (r *boltDBPubSubRep) FetchNodes(_ context.Context, host string) ([]*pubsubmodel.Node, error) {
	var retVal []*pubsubmodel.Node

	op := iterKeysOp{
		tx:     r.tx,
		bucket: pubSubNodeBucketKey(host),
		iterFn: func(_, b []byte) error {
			var node pubsubmodel.Node
			if err := proto.Unmarshal(b, &node); err != nil {
				return err
			}
			retVal = append(retVal, &node)
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func (r *boltDBPubSubRep) NodeExists(_ context.Context, host, name string) (bool, error) {
	op := keyExistsOp{
		tx:     r.tx,
		bucket: pubSubNodeBucketKey(host),
		key:    name,
	}
	return op.do(), nil
}

func (r *boltDBPubSubRep) DeleteNode(_ context.Context, host, name string) error {
	op := delKeyOp{
		tx:     r.tx,
		bucket: pubSubNodeBucketKey(host),
		key:    name,
	}
	return op.do()
}

func (r *boltDBPubSubRep) DeleteNodes(_ context.Context, host string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: pubSubNodeBucketKey(host),
	}
	return op.do()
}

func (r *boltDBPubSubRep) UpsertNodeAffiliation(_ context.Context, aff *pubsubmodel.Affiliation, host, name string) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: pubSubAffiliationsBucketKey(host, name),
		key:    aff.Jid,
		obj:    aff,
	}
	return op.do()
}

func (r *boltDBPubSubRep) FetchNodeAffiliation(_ context.Context, jid, host, name string) (*pubsubmodel.Affiliation, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: pubSubAffiliationsBucketKey(host, name),
		key:    jid,
		obj:    &pubsubmodel.Affiliation{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*pubsubmodel.Affiliation), nil
	default:
		return nil, nil
	}
}

func (r *boltDBPubSubRep) FetchNodeAffiliations(_ context.Context, host, name string) ([]*pubsubmodel.Affiliation, error) {
	var retVal []*pubsubmodel.Affiliation

	op := iterKeysOp{
		tx:     r.tx,
		bucket: pubSubAffiliationsBucketKey(host, name),
		iterFn: func(_, b []byte) error {
			var aff pubsubmodel.Affiliation
			if err := proto.Unmarshal(b, &aff); err != nil {
				return err
			}
			retVal = append(retVal, &aff)
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func (r *boltDBPubSubRep) DeleteNodeAffiliation(_ context.Context, jid, host, name string) error {
	op := delKeyOp{
		tx:     r.tx,
		bucket: pubSubAffiliationsBucketKey(host, name),
		key:    jid,
	}
	return op.do()
}

func (r *boltDBPubSubRep) DeleteNodeAffiliations(_ context.Context, host, name string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: pubSubAffiliationsBucketKey(host, name),
	}
	return op.do()
}

func (r *boltDBPubSubRep) UpsertNodeSubscription(_ context.Context, sub *pubsubmodel.Subscription, host, name string) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: pubSubSubscriptionsBucketKey(host, name),
		key:    sub.Jid,
		obj:    sub,
	}
	return op.do()
}

func (r *boltDBPubSubRep) FetchNodeSubscription(_ context.Context, jid, host, name string) (*pubsubmodel.Subscription, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: pubSubSubscriptionsBucketKey(host, name),
		key:    jid,
		obj:    &pubsubmodel.Subscription{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*pubsubmodel.Subscription), nil
	default:
		return nil, nil
	}
}

func (r *boltDBPubSubRep) FetchNodeSubscriptions(_ context.Context, host, name string) ([]*pubsubmodel.Subscription, error) {
	var retVal []*pubsubmodel.Subscription

	op := iterKeysOp{
		tx:     r.tx,
		bucket: pubSubSubscriptionsBucketKey(host, name),
		iterFn: func(_, b []byte) error {
			var sub pubsubmodel.Subscription
			if err := proto.Unmarshal(b, &sub); err != nil {
				return err
			}
			retVal = append(retVal, &sub)
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func (r *boltDBPubSubRep) DeleteNodeSubscription(_ context.Context, jid, host, name string) error {
	op := delKeyOp{
		tx:     r.tx,
		bucket: pubSubSubscriptionsBucketKey(host, name),
		key:    jid,
	}
	return op.do()
}

func (r *boltDBPubSubRep) DeleteNodeSubscriptions(_ context.Context, host, name string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: pubSubSubscriptionsBucketKey(host, name),
	}
	return op.do()
}

func (r *boltDBPubSubRep) InsertNodeItem(_ context.Context, item *pubsubmodel.Item, host, name string) error {
	op := insertSeqOp{
		tx:     r.tx,
		bucket: pubSubItemsBucketKey(host, name),
		obj:    item,
	}
	return op.do()
}

func (r *boltDBPubSubRep) FetchNodeItems(_ context.Context, host, name string) ([]*pubsubmodel.Item, error) {
	var retVal []*pubsubmodel.Item

	op := iterKeysOp{
		tx:     r.tx,
		bucket: pubSubItemsBucketKey(host, name),
		iterFn: func(_, b []byte) error {
			var itm pubsubmodel.Item
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

func (r *boltDBPubSubRep) DeleteOldestNodeItems(_ context.Context, host, name string, maxItems int) error {
	bucketID := pubSubItemsBucketKey(host, name)

	b := r.tx.Bucket([]byte(bucketID))
	if b == nil {
		return nil
	}
	// count items
	var count int

	c := b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		count++
	}
	if count < maxItems {
		return nil
	}
	// store old value keys
	var oldKeys [][]byte

	c = b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		if count <= maxItems {
			break
		}
		count--
		oldKeys = append(oldKeys, k)
	}
	// delete old values
	for _, k := range oldKeys {
		if err := b.Delete(k); err != nil {
			return err
		}
	}
	return nil
}

func (r *boltDBPubSubRep) DeleteNodeItems(_ context.Context, host, name string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: pubSubItemsBucketKey(host, name),
	}
	return op.do()
}

func pubSubNodeBucketKey(host string) string {
	return fmt.Sprintf("pubsub:node:%s", host)
}

func pubSubAffiliationsBucketKey(host, name string) string {
	return fmt.Sprintf("pubsub:aff:%s:%s", host, name)
}

func pubSubSubscriptionsBucketKey(host, name string) string {
	return fmt.Sprintf("pubsub:sub:%s:%s", host, name)
}

func pubSubItemsBucketKey(host, name string) string {
	return fmt.Sprintf("pubsub:itm:%s:%s", host, name)
}

// UpsertNode inserts a node entity into storage, or updates it if was previously inserted.
func (r *Repository) UpsertNode(ctx context.Context, node *pubsubmodel.Node) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).UpsertNode(ctx, node)
	})
}

// FetchNodes retrieves from storage all node entities associated to a host.
func (r *Repository) FetchNodes(ctx context.Context, host string) (nodes []*pubsubmodel.Node, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		nodes, err = newPubSubRep(tx).FetchNodes(ctx, host)
		return err
	})
	return
}

// NodeExists tells whether a node for a given host exists.
func (r *Repository) NodeExists(ctx context.Context, host, name string) (ok bool, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		ok, err = newPubSubRep(tx).NodeExists(ctx, host, name)
		return err
	})
	return
}

// DeleteNode deletes a pubsub node from storage.
func (r *Repository) DeleteNode(ctx context.Context, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteNode(ctx, host, name)
	})
}

// DeleteNodes deletes all nodes associated to a host from storage.
func (r *Repository) DeleteNodes(ctx context.Context, host string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteNodes(ctx, host)
	})
}

// UpsertNodeAffiliation inserts a new pubsub node affiliation into storage, or updates it if previously inserted.
func (r *Repository) UpsertNodeAffiliation(ctx context.Context, affiliation *pubsubmodel.Affiliation, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).UpsertNodeAffiliation(ctx, affiliation, host, name)
	})
}

// FetchNodeAffiliations retrieves all affiliations associated to a node.
func (r *Repository) FetchNodeAffiliations(ctx context.Context, host, name string) (affiliations []*pubsubmodel.Affiliation, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		affiliations, err = newPubSubRep(tx).FetchNodeAffiliations(ctx, host, name)
		return err
	})
	return
}

// DeleteNodeAffiliation deletes a pubsub node affiliation from storage.
func (r *Repository) DeleteNodeAffiliation(ctx context.Context, jid, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteNodeAffiliation(ctx, jid, host, name)
	})
}

// DeleteNodeAffiliations deletes all affiliations associated to a node.
func (r *Repository) DeleteNodeAffiliations(ctx context.Context, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteNodeAffiliations(ctx, host, name)
	})
}

// UpsertNodeSubscription inserts a new pubsub node subscription into storage, or updates it if previously inserted.
func (r *Repository) UpsertNodeSubscription(ctx context.Context, subscription *pubsubmodel.Subscription, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).UpsertNodeSubscription(ctx, subscription, host, name)
	})
}

// FetchNodeSubscriptions retrieves all subscriptions associated to a node.
func (r *Repository) FetchNodeSubscriptions(ctx context.Context, host, name string) (subscriptions []*pubsubmodel.Subscription, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		subscriptions, err = newPubSubRep(tx).FetchNodeSubscriptions(ctx, host, name)
		return err
	})
	return
}

// DeleteNodeSubscription deletes a pubsub node subscription from storage.
func (r *Repository) DeleteNodeSubscription(ctx context.Context, jid, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteNodeSubscription(ctx, jid, host, name)
	})
}

// DeleteNodeSubscriptions deletes all subscriptions associated to a node.
func (r *Repository) DeleteNodeSubscriptions(ctx context.Context, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteNodeSubscriptions(ctx, host, name)
	})
}

// InsertNodeItem inserts a new pubsub node item into storage.
func (r *Repository) InsertNodeItem(ctx context.Context, item *pubsubmodel.Item, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).InsertNodeItem(ctx, item, host, name)
	})
}

// FetchNodeItems retrieves all items associated to a node.
func (r *Repository) FetchNodeItems(ctx context.Context, host, name string) (items []*pubsubmodel.Item, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		items, err = newPubSubRep(tx).FetchNodeItems(ctx, host, name)
		return err
	})
	return
}

// DeleteOldestNodeItems deletes the oldest items associated to a node.
func (r *Repository) DeleteOldestNodeItems(ctx context.Context, host, name string, maxItems int) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteOldestNodeItems(ctx, host, name, maxItems)
	})
}

// DeleteNodeItems deletes all items associated to a node.
func (r *Repository) DeleteNodeItems(ctx context.Context, host, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPubSubRep(tx).DeleteNodeItems(ctx, host, name)
	})
}
