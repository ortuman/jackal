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

package measuredrepository

import (
	"context"
	"time"

	lastmodel "github.com/ortuman/jackal/pkg/model/last"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type measuredLastRep struct {
	rep  repository.Last
	inTx bool
}

func (m *measuredLastRep) UpsertLast(ctx context.Context, last *lastmodel.Last) error {
	t0 := time.Now()
	err := m.rep.UpsertLast(ctx, last)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredLastRep) FetchLast(ctx context.Context, username string) (last *lastmodel.Last, err error) {
	t0 := time.Now()
	last, err = m.rep.FetchLast(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredLastRep) DeleteLast(ctx context.Context, username string) error {
	t0 := time.Now()
	err := m.rep.DeleteLast(ctx, username)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}
