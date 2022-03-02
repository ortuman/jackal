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

	"github.com/jackal-xmpp/stravaganza"
	bolt "go.etcd.io/bbolt"
)

type boltDBPrivateRep struct {
	tx *bolt.Tx
}

func newPrivateRep(tx *bolt.Tx) *boltDBPrivateRep {
	return &boltDBPrivateRep{tx: tx}
}

func (r *boltDBPrivateRep) FetchPrivate(_ context.Context, namespace, username string) (stravaganza.Element, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: privateBucketKey(username),
		key:    namespace,
		obj:    stravaganza.EmptyElement(),
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(stravaganza.Element), nil
	default:
		return nil, nil
	}
}

func (r *boltDBPrivateRep) UpsertPrivate(_ context.Context, private stravaganza.Element, namespace, username string) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: privateBucketKey(username),
		key:    namespace,
		obj:    private,
	}
	return op.do()
}

func (r *boltDBPrivateRep) DeletePrivates(_ context.Context, username string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: privateBucketKey(username),
	}
	return op.do()
}

func privateBucketKey(username string) string {
	return fmt.Sprintf("prv:%s", username)
}

// FetchPrivate satisfies repository.Private interface.
func (r *Repository) FetchPrivate(ctx context.Context, namespace, username string) (prv stravaganza.Element, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		prv, err = newPrivateRep(tx).FetchPrivate(ctx, namespace, username)
		return err
	})
	return
}

// UpsertPrivate satisfies repository.Private interface.
func (r *Repository) UpsertPrivate(ctx context.Context, private stravaganza.Element, namespace, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPrivateRep(tx).UpsertPrivate(ctx, private, namespace, username)
	})
}

// DeletePrivates satisfies repository.Private interface.
func (r *Repository) DeletePrivates(ctx context.Context, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newPrivateRep(tx).DeletePrivates(ctx, username)
	})
}
