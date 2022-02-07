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

const vCardKey = "vc"

type vCardCodec struct {
	val stravaganza.Element
}

func (c *vCardCodec) encode(i interface{}) ([]byte, error) {
	el := i.(stravaganza.Element)
	return proto.Marshal(el.Proto())
}

func (c *vCardCodec) decode(b []byte) error {
	sb, err := stravaganza.NewBuilderFromBinary(b)
	if err != nil {
		return err
	}
	c.val = sb.Build()
	return nil
}

func (c *vCardCodec) value() interface{} {
	return c.val
}

type cachedVCardRep struct {
	c   Cache
	rep repository.VCard
}

func (c *cachedVCardRep) UpsertVCard(ctx context.Context, vCard stravaganza.Element, username string) error {
	op := updateOp{
		c:              c.c,
		namespace:      vCardNS(username),
		invalidateKeys: []string{vCardKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertVCard(ctx, vCard, username)
		},
	}
	return op.do(ctx)
}

func (c *cachedVCardRep) FetchVCard(ctx context.Context, username string) (stravaganza.Element, error) {
	op := fetchOp{
		c:         c.c,
		namespace: vCardNS(username),
		key:       vCardKey,
		codec:     &vCardCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchVCard(ctx, username)
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

func (c *cachedVCardRep) DeleteVCard(ctx context.Context, username string) error {
	op := updateOp{
		c:              c.c,
		namespace:      vCardNS(username),
		invalidateKeys: []string{vCardKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteVCard(ctx, username)
		},
	}
	return op.do(ctx)
}

func vCardNS(username string) string {
	return fmt.Sprintf("vc:%s", username)
}
