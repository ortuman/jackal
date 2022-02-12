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
	capsmodel "github.com/ortuman/jackal/pkg/model/caps"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const capsKey = "caps"

type capsCodec struct {
	val *capsmodel.Capabilities
}

func (c *capsCodec) encode(i interface{}) ([]byte, error) {
	return proto.Marshal(i.(*capsmodel.Capabilities))
}

func (c *capsCodec) decode(b []byte) error {
	var caps capsmodel.Capabilities
	if err := proto.Unmarshal(b, &caps); err != nil {
		return err
	}
	c.val = &caps
	return nil
}

func (c *capsCodec) value() interface{} {
	return c.val
}

type cachedCapsRep struct {
	c   Cache
	rep repository.Capabilities
}

func (c *cachedCapsRep) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error {
	op := updateOp{
		c:              c.c,
		namespace:      capsNS(caps.Node, caps.Ver),
		invalidateKeys: []string{capsKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertCapabilities(ctx, caps)
		},
	}
	return op.do(ctx)
}

func (c *cachedCapsRep) CapabilitiesExist(ctx context.Context, node, ver string) (bool, error) {
	op := existsOp{
		c:         c.c,
		namespace: capsNS(node, ver),
		key:       capsKey,
		missFn: func(ctx context.Context) (bool, error) {
			return c.rep.CapabilitiesExist(ctx, node, ver)
		},
	}
	return op.do(ctx)
}

func (c *cachedCapsRep) FetchCapabilities(ctx context.Context, node, ver string) (*capsmodel.Capabilities, error) {
	op := fetchOp{
		c:         c.c,
		namespace: capsNS(node, ver),
		key:       capsKey,
		codec:     &capsCodec{},
		missFn: func(ctx context.Context) (interface{}, error) {
			return c.rep.FetchCapabilities(ctx, node, ver)
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*capsmodel.Capabilities), nil
	}
	return nil, nil
}

func capsNS(node, ver string) string {
	return fmt.Sprintf("caps:%s:%s", node, ver)
}
