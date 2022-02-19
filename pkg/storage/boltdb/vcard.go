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

const vCardKey = "vcard"

type boltDBVCardRep struct {
	tx *bolt.Tx
}

func newVCardRep(tx *bolt.Tx) *boltDBVCardRep {
	return &boltDBVCardRep{tx: tx}
}

func (r *boltDBVCardRep) UpsertVCard(_ context.Context, vCard stravaganza.Element, username string) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: vCardBucketKey(username),
		key:    vCardKey,
		obj:    vCard,
	}
	return op.do()
}

func (r *boltDBVCardRep) FetchVCard(_ context.Context, username string) (stravaganza.Element, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: vCardBucketKey(username),
		key:    vCardKey,
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

func (r *boltDBVCardRep) DeleteVCard(_ context.Context, username string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: vCardBucketKey(username),
	}
	return op.do()
}

func vCardBucketKey(username string) string {
	return fmt.Sprintf("vcard:%s", username)
}

// UpsertVCard satisfies repository.VCard interface.
func (r *Repository) UpsertVCard(ctx context.Context, vCard stravaganza.Element, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newVCardRep(tx).UpsertVCard(ctx, vCard, username)
	})
}

// FetchVCard satisfies repository.VCard interface.
func (r *Repository) FetchVCard(ctx context.Context, username string) (vc stravaganza.Element, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		vc, err = newVCardRep(tx).FetchVCard(ctx, username)
		return err
	})
	return
}

// DeleteVCard satisfies repository.VCard interface.
func (r *Repository) DeleteVCard(ctx context.Context, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newVCardRep(tx).DeleteVCard(ctx, username)
	})
}
