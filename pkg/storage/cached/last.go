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

	"github.com/go-kit/log"
	"github.com/ortuman/jackal/pkg/model"

	lastmodel "github.com/ortuman/jackal/pkg/model/last"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const lastKey = "lst"

type cachedLastRep struct {
	c      Cache
	rep    repository.Last
	logger log.Logger
}

func (c *cachedLastRep) UpsertLast(ctx context.Context, last *lastmodel.Last) error {
	op := updateOp{
		c:              c.c,
		namespace:      lastNS(last.Username),
		invalidateKeys: []string{lastKey},
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
		codec:     &lastmodel.Last{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return c.rep.FetchLast(ctx, username)
		},
		logger: c.logger,
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
		c:              c.c,
		namespace:      lastNS(username),
		invalidateKeys: []string{lastKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteLast(ctx, username)
		},
	}
	return op.do(ctx)
}

func lastNS(username string) string {
	return fmt.Sprintf("lst:%s", username)
}
