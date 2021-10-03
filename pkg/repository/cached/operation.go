// Copyright 2021 The jackal Authors
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
	"errors"

	"github.com/golang/protobuf/proto"
)

var errInvalidObject = errors.New("invalid cacheable object")

type writeOp struct {
	c   Cache
	key string
	fn  func(context.Context) error
}

func (op *writeOp) perform(ctx context.Context) error {
	if err := op.fn(ctx); err != nil {
		return err
	}
	return op.c.Del(ctx, op.key)
}

type readOp struct {
	c   Cache
	key string
	fn  func(context.Context) (interface{}, error)
	obj interface{}

	fetched bool
}

func (op *readOp) perform(ctx context.Context) error {
	if err := op.fetchCached(ctx); err != nil {
		return nil
	}
	if op.fetched {
		return nil
	}
	obj, err := op.fn(ctx)
	if err != nil {
		return err
	}
	if obj == nil {
		return nil
	}
	op.obj = obj
	op.fetched = true
	return op.storeCached(ctx)
}

func (op *readOp) fetchCached(ctx context.Context) error {
	b, err := op.c.Fetch(ctx, op.key)
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	m, ok := op.obj.(proto.Message)
	if !ok {
		return errInvalidObject
	}
	if err := proto.Unmarshal(b, m); err != nil {
		return err
	}
	op.fetched = true
	return nil
}

func (op *readOp) storeCached(ctx context.Context) error {
	m, ok := op.obj.(proto.Message)
	if !ok {
		return errInvalidObject
	}
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	return op.c.Store(ctx, op.key, b)
}

type existsOp struct {
	c   Cache
	key string
	fn  func(context.Context) (bool, error)

	exists bool
}

func (op *existsOp) perform(ctx context.Context) error {
	exists, err := op.c.Exists(ctx, op.key)
	if err != nil {
		return err
	}
	if exists {
		op.exists = true
		return nil
	}
	exists, err = op.fn(ctx)
	if err != nil {
		return err
	}
	op.exists = exists
	return nil
}
