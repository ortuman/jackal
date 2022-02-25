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

package measuredrepository

import (
	"context"
	"time"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type measuredOfflineRep struct {
	rep  repository.Offline
	inTx bool
}

func (m *measuredOfflineRep) InsertOfflineMessage(ctx context.Context, message *stravaganza.Message, username string) error {
	t0 := time.Now()
	err := m.rep.InsertOfflineMessage(ctx, message, username)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}

func (m *measuredOfflineRep) CountOfflineMessages(ctx context.Context, username string) (int, error) {
	t0 := time.Now()
	count, err := m.rep.CountOfflineMessages(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return count, err
}

func (m *measuredOfflineRep) FetchOfflineMessages(ctx context.Context, username string) ([]*stravaganza.Message, error) {
	t0 := time.Now()
	ms, err := m.rep.FetchOfflineMessages(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return ms, err
}

func (m *measuredOfflineRep) DeleteOfflineMessages(ctx context.Context, username string) error {
	t0 := time.Now()
	err := m.rep.DeleteOfflineMessages(ctx, username)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil, m.inTx)
	return err
}
