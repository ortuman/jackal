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

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/repository"
)

type measuredUserRep struct {
	rep repository.User
}

func (m *measuredUserRep) UpsertUser(ctx context.Context, user *model.User) (err error) {
	t0 := time.Now()
	err = m.rep.UpsertUser(ctx, user)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredUserRep) DeleteUser(ctx context.Context, username string) (err error) {
	t0 := time.Now()
	err = m.rep.DeleteUser(ctx, username)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredUserRep) FetchUser(ctx context.Context, username string) (usr *model.User, err error) {
	t0 := time.Now()
	usr, err = m.rep.FetchUser(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredUserRep) UserExists(ctx context.Context, username string) (ok bool, err error) {
	t0 := time.Now()
	ok, err = m.rep.UserExists(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return
}
