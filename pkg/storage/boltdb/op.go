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
	"fmt"

	bolt "go.etcd.io/bbolt"

	"github.com/ortuman/jackal/pkg/model"
)

type upsertKeyOp struct {
	tx     *bolt.Tx
	bucket string
	key    string
	obj    model.Codec
}

func (op upsertKeyOp) do() error {
	b, err := op.tx.CreateBucketIfNotExists([]byte(op.bucket))
	if err != nil {
		return err
	}
	p, err := op.obj.MarshalBinary()
	if err != nil {
		return err
	}
	return b.Put([]byte(op.key), p)
}

type insertSeqOp struct {
	tx     *bolt.Tx
	bucket string
	obj    model.Codec
}

func (op insertSeqOp) do() error {
	b, err := op.tx.CreateBucketIfNotExists([]byte(op.bucket))
	if err != nil {
		return err
	}
	p, err := op.obj.MarshalBinary()
	if err != nil {
		return err
	}
	seq, err := b.NextSequence()
	if err != nil {
		return err
	}
	k := fmt.Sprintf("%d", seq)
	return b.Put([]byte(k), p)
}

type delBucketOp struct {
	tx     *bolt.Tx
	bucket string
}

func (op delBucketOp) do() error {
	return op.tx.DeleteBucket([]byte(op.bucket))
}

type delKeyOp struct {
	tx     *bolt.Tx
	bucket string
	key    string
}

func (op delKeyOp) do() error {
	b := op.tx.Bucket([]byte(op.bucket))
	if b == nil {
		return nil
	}
	return b.Delete([]byte(op.key))
}

type bucketExistsOp struct {
	tx     *bolt.Tx
	bucket string
}

func (op bucketExistsOp) do() bool {
	return op.tx.Bucket([]byte(op.bucket)) != nil
}

type fetchKeyOp struct {
	tx     *bolt.Tx
	bucket string
	key    string
	obj    model.Codec
}

func (op fetchKeyOp) do() (model.Codec, error) {
	b := op.tx.Bucket([]byte(op.bucket))
	if b == nil {
		return nil, nil
	}
	data := b.Get([]byte(op.key))
	if data == nil {
		return nil, nil
	}
	if err := op.obj.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	return op.obj, nil
}

type countKeysOp struct {
	tx     *bolt.Tx
	bucket string
}

func (op countKeysOp) do() (int, error) {
	b := op.tx.Bucket([]byte(op.bucket))
	if b == nil {
		return 0, nil
	}
	var retVal int

	c := b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		retVal++
	}
	return retVal, nil
}

type iterKeysOp struct {
	tx     *bolt.Tx
	bucket string
	iterFn func(k, b []byte) error
}

func (op iterKeysOp) do() error {
	b := op.tx.Bucket([]byte(op.bucket))
	if b == nil {
		return nil
	}
	c := b.Cursor()

	for k, v := c.First(); k != nil; k, v = c.Next() {
		if err := op.iterFn(k, v); err != nil {
			return err
		}
	}
	return nil
}
