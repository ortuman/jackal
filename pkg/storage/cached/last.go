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
	"fmt"

	"github.com/golang/protobuf/proto"
	lastmodel "github.com/ortuman/jackal/pkg/model/last"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const lastKey = "lst"

type lastCodec struct {
	val *lastmodel.Last
}

func (c *lastCodec) encode(i interface{}) ([]byte, error) {
	return proto.Marshal(i.(*lastmodel.Last))
}

func (c *lastCodec) decode(b []byte) error {
	var last lastmodel.Last
	if err := proto.Unmarshal(b, &last); err != nil {
		return err
	}
	c.val = &last
	return nil
}

func (c *lastCodec) value() interface{} {
	return c.val
}

type cachedLastRep struct {
	c   Cache
	rep repository.Last
}

func (c *cachedLastRep) UpsertLast(ctx context.Context, last *lastmodel.Last) error {
	op := updateOp{
		c:           c.c,
		namespace:   lastNS(last.Username),
		invalidKeys: []string{lastKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertLast(ctx, last)
		},
	}
	return op.do(ctx)
}

func (c *cachedLastRep) FetchLast(ctx context.Context, username string) (*lastmodel.Last, error) {
	op := fetchOp{
		c:         c.c,
		namespace: lastNS(username),
		key:       lastKey,
		codec:     &lastCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchLast(ctx, username)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*lastmodel.Last), nil
	}
	return nil, nil
}

func (c *cachedLastRep) DeleteLast(ctx context.Context, username string) error {
	op := updateOp{
		c:           c.c,
		namespace:   lastNS(username),
		invalidKeys: []string{lastKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteLast(ctx, username)
		},
	}
	return op.do(ctx)
}

func lastNS(username string) string {
	return fmt.Sprintf("lst:%s", username)
}
