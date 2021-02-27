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

	capsmodel "github.com/ortuman/jackal/model/caps"
	"github.com/ortuman/jackal/repository"
)

type measuredCapabilitiesRep struct {
	rep repository.Capabilities
}

func (m *measuredCapabilitiesRep) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) (err error) {
	t0 := time.Now()
	err = m.rep.UpsertCapabilities(ctx, caps)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return
}

func (m *measuredCapabilitiesRep) FetchCapabilities(ctx context.Context, node, ver string) (caps *capsmodel.Capabilities, err error) {
	t0 := time.Now()
	caps, err = m.rep.FetchCapabilities(ctx, node, ver)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return
}
