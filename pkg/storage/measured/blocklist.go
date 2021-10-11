// Copyright 2020 The jackal Authors
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

package measuredrepository

import (
	"context"
	"time"

	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type measuredBlockListRep struct {
	rep repository.BlockList
}

func (m *measuredBlockListRep) UpsertBlockListItem(ctx context.Context, item *blocklistmodel.Item) (err error) {
	t0 := time.Now()
	err = m.rep.UpsertBlockListItem(ctx, item)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredBlockListRep) DeleteBlockListItem(ctx context.Context, item *blocklistmodel.Item) (err error) {
	t0 := time.Now()
	err = m.rep.DeleteBlockListItem(ctx, item)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredBlockListRep) FetchBlockListItems(ctx context.Context, username string) (blockList []blocklistmodel.Item, err error) {
	t0 := time.Now()
	blockList, err = m.rep.FetchBlockListItems(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredBlockListRep) DeleteBlockListItems(ctx context.Context, username string) (err error) {
	t0 := time.Now()
	err = m.rep.DeleteBlockListItems(ctx, username)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return
}
