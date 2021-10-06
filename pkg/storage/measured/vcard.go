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

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

type measuredVCardRep struct {
	rep repository.VCard
}

func (m *measuredVCardRep) UpsertVCard(ctx context.Context, vCard stravaganza.Element, username string) error {
	t0 := time.Now()
	err := m.rep.UpsertVCard(ctx, vCard, username)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return err
}

func (m *measuredVCardRep) FetchVCard(ctx context.Context, username string) (stravaganza.Element, error) {
	t0 := time.Now()
	vc, err := m.rep.FetchVCard(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return vc, err
}

func (m *measuredVCardRep) DeleteVCard(ctx context.Context, username string) (err error) {
	t0 := time.Now()
	err = m.rep.DeleteVCard(ctx, username)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return
}
