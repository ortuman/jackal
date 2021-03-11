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
	"github.com/ortuman/jackal/repository"
)

type measuredPrivateRep struct {
	rep repository.Private
}

func (m *measuredPrivateRep) FetchPrivate(ctx context.Context, namespace, username string) (private stravaganza.Element, err error) {
	t0 := time.Now()
	private, err = m.rep.FetchPrivate(ctx, namespace, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredPrivateRep) UpsertPrivate(ctx context.Context, private stravaganza.Element, namespace, username string) (err error) {
	t0 := time.Now()
	err = m.rep.UpsertPrivate(ctx, private, namespace, username)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredPrivateRep) DeletePrivates(ctx context.Context, username string) (err error) {
	t0 := time.Now()
	err = m.rep.DeletePrivates(ctx, username)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return
}
