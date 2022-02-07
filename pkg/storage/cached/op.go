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
)

type codec interface {
	encode(i interface{}) ([]byte, error)
	decode([]byte) error
	value() interface{}
}

type existsOp struct {
	c         Cache
	namespace string
	key       string
	missFn    func(context.Context) (bool, error)
}

func (op existsOp) do(ctx context.Context) (bool, error) {
	ok, err := op.c.HasKey(ctx, op.namespace, op.key)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	return op.missFn(ctx)
}

type updateOp struct {
	c              Cache
	namespace      string
	invalidateKeys []string
	updateFn       func(context.Context) error
}

func (op updateOp) do(ctx context.Context) error {
	switch {
	case len(op.invalidateKeys) > 0:
		if err := op.c.Del(ctx, op.namespace, op.invalidateKeys...); err != nil {
			return err
		}

	default:
		if err := op.c.DelNS(ctx, op.namespace); err != nil {
			return err
		}
	}
	return op.updateFn(ctx)
}

type fetchOp struct {
	c         Cache
	namespace string
	key       string
	codec     codec
	missFn    func(context.Context) (interface{}, error)
}

func (op fetchOp) do(ctx context.Context) (interface{}, error) {
	b, err := op.c.Get(ctx, op.namespace, op.key)
	if err != nil {
		return nil, err
	}
	if b == nil {
		obj, err := op.missFn(ctx)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			return nil, nil
		}
		b, err = op.codec.encode(obj)
		if err != nil {
			return nil, err
		}
		if err := op.c.Put(ctx, op.namespace, op.key, b); err != nil {
			return nil, err
		}
		return obj, nil
	}
	if err := op.codec.decode(b); err != nil {
		return nil, err
	}
	return op.codec.value(), nil
}
