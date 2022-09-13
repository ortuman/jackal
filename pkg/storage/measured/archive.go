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

	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type measuredArchiveRep struct {
	rep  repository.Archive
	inTx bool
}

func (m *measuredArchiveRep) InsertArchiveMessage(ctx context.Context, message *archivemodel.Message) error {
	t0 := time.Now()
	err := m.rep.InsertArchiveMessage(ctx, message)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredArchiveRep) FetchArchiveMetadata(ctx context.Context, archiveID string) (metadata *archivemodel.Metadata, err error) {
	t0 := time.Now()
	metadata, err = m.rep.FetchArchiveMetadata(ctx, archiveID)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredArchiveRep) FetchArchiveMessages(ctx context.Context, f *archivemodel.Filters, archiveID string) (messages []*archivemodel.Message, err error) {
	t0 := time.Now()
	messages, err = m.rep.FetchArchiveMessages(ctx, f, archiveID)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return
}

func (m *measuredArchiveRep) DeleteArchiveOldestMessages(ctx context.Context, archiveID string, maxElements int) error {
	t0 := time.Now()
	err := m.rep.DeleteArchiveOldestMessages(ctx, archiveID, maxElements)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredArchiveRep) DeleteArchive(ctx context.Context, archiveID string) error {
	t0 := time.Now()
	err := m.rep.DeleteArchive(ctx, archiveID)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}
