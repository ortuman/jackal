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

	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type privateCodec struct {
	val stravaganza.Element
}

func (c *privateCodec) encode(i interface{}) ([]byte, error) {
	el := i.(stravaganza.Element)
	return proto.Marshal(el.Proto())
}

func (c *privateCodec) decode(b []byte) error {
	sb, err := stravaganza.NewBuilderFromBinary(b)
	if err != nil {
		return err
	}
	c.val = sb.Build()
	return nil
}

func (c *privateCodec) value() interface{} {
	return c.val
}

type cachedPrivateRep struct {
	c   Cache
	rep repository.Private
}

func (c *cachedPrivateRep) FetchPrivate(ctx context.Context, namespace, username string) (stravaganza.Element, error) {
	op := fetchOp{
		c:         c.c,
		namespace: privateNS(username),
		key:       namespace,
		codec:     &privateCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchPrivate(ctx, namespace, username)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(stravaganza.Element), nil
	}
	return nil, nil
}

func (c *cachedPrivateRep) UpsertPrivate(ctx context.Context, private stravaganza.Element, namespace, username string) error {
	op := updateOp{
		c:              c.c,
		namespace:      privateNS(username),
		invalidateKeys: []string{namespace},
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertPrivate(ctx, private, namespace, username)
		},
	}
	return op.do(ctx)
}

func (c *cachedPrivateRep) DeletePrivates(ctx context.Context, username string) error {
	op := updateOp{
		c:         c.c,
		namespace: privateNS(username),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeletePrivates(ctx, username)
		},
	}
	return op.do(ctx)
}

func privateNS(username string) string {
	return fmt.Sprintf("prv:%s", username)
}
