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

	capsmodel "github.com/ortuman/jackal/pkg/model/caps"
	bolt "go.etcd.io/bbolt"
)

const capsKey = "caps"

type boltDBCapsRep struct {
	tx *bolt.Tx
}

func newCapsRep(tx *bolt.Tx) *boltDBCapsRep {
	return &boltDBCapsRep{tx: tx}
}

func (r *boltDBCapsRep) UpsertCapabilities(_ context.Context, caps *capsmodel.Capabilities) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: capsBucketKey(caps.Node, caps.Ver),
		key:    capsKey,
		obj:    caps,
	}
	return op.do()
}

func (r *boltDBCapsRep) CapabilitiesExist(_ context.Context, node, ver string) (bool, error) {
	op := bucketExistsOp{
		tx:     r.tx,
		bucket: capsBucketKey(node, ver),
	}
	return op.do(), nil
}

func (r *boltDBCapsRep) FetchCapabilities(_ context.Context, node, ver string) (*capsmodel.Capabilities, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: capsBucketKey(node, ver),
		key:    capsKey,
		obj:    &capsmodel.Capabilities{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*capsmodel.Capabilities), nil
	default:
		return nil, nil
	}
}

func capsBucketKey(node, ver string) string {
	return fmt.Sprintf("caps:%s:%s", node, ver)
}

// UpsertCapabilities satisfies repository.Capabilities interface.
func (r *Repository) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newCapsRep(tx).UpsertCapabilities(ctx, caps)
	})
}

// CapabilitiesExist tells whether node+ver capabilities have been already registered.
func (r *Repository) CapabilitiesExist(ctx context.Context, node, ver string) (ok bool, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		ok, err = newCapsRep(tx).CapabilitiesExist(ctx, node, ver)
		return err
	})
	return
}

// FetchCapabilities fetches capabilities associated to a given node+ver pair.
func (r *Repository) FetchCapabilities(ctx context.Context, node, ver string) (caps *capsmodel.Capabilities, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		caps, err = newCapsRep(tx).FetchCapabilities(ctx, node, ver)
		return err
	})
	return
}
