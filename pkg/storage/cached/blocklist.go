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

	"github.com/ortuman/jackal/pkg/model"

	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const blockListItems = "items"

type cachedBlockListRep struct {
	c   Cache
	rep repository.BlockList
}

func (c *cachedBlockListRep) UpsertBlockListItem(ctx context.Context, item *blocklistmodel.Item) error {
	op := updateOp{
		c:              c.c,
		namespace:      blockListNS(item.Username),
		invalidateKeys: []string{blockListItems},
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertBlockListItem(ctx, item)
		},
	}
	return op.do(ctx)
}

func (c *cachedBlockListRep) DeleteBlockListItem(ctx context.Context, item *blocklistmodel.Item) error {
	op := updateOp{
		c:              c.c,
		namespace:      blockListNS(item.Username),
		invalidateKeys: []string{blockListItems},
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteBlockListItem(ctx, item)
		},
	}
	return op.do(ctx)
}

func (c *cachedBlockListRep) FetchBlockListItems(ctx context.Context, username string) ([]*blocklistmodel.Item, error) {
	op := fetchOp{
		c:         c.c,
		namespace: blockListNS(username),
		key:       blockListItems,
		codec:     &blocklistmodel.Items{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			items, err := c.rep.FetchBlockListItems(ctx, username)
			if err != nil {
				return nil, err
			}
			return &blocklistmodel.Items{Items: items}, nil
		},
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*blocklistmodel.Items).Items, nil
	}
	return nil, nil
}

func (c *cachedBlockListRep) DeleteBlockListItems(ctx context.Context, username string) error {
	op := updateOp{
		c:         c.c,
		namespace: blockListNS(username),
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteBlockListItems(ctx, username)
		},
	}
	return op.do(ctx)
}

func blockListNS(username string) string {
	return fmt.Sprintf("bl:%s", username)
}
