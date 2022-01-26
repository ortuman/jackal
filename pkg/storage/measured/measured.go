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
	"strconv"

	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/ortuman/jackal/pkg/storage/repository"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	upsertOp = "upsert"
	fetchOp  = "fetch"
	deleteOp = "delete"
	lockOp   = "lock"
)

// Measured is measured Repository implementation.
type Measured struct {
	measuredUserRep
	measuredLastRep
	measuredCapabilitiesRep
	measuredOfflineRep
	measuredBlockListRep
	measuredPrivateRep
	measuredRosterRep
	measuredVCardRep
	measuredLocker
	rep repository.Repository
}

// New returns a new initialized Measured repository.
func New(rep repository.Repository) repository.Repository {
	return &Measured{
		measuredUserRep:         measuredUserRep{rep: rep},
		measuredLastRep:         measuredLastRep{rep: rep},
		measuredCapabilitiesRep: measuredCapabilitiesRep{rep: rep},
		measuredOfflineRep:      measuredOfflineRep{rep: rep},
		measuredBlockListRep:    measuredBlockListRep{rep: rep},
		measuredPrivateRep:      measuredPrivateRep{rep: rep},
		measuredRosterRep:       measuredRosterRep{rep: rep},
		measuredVCardRep:        measuredVCardRep{rep: rep},
		measuredLocker:          measuredLocker{rep: rep},
		rep:                     rep,
	}
}

// InTransaction generates a repository transaction and completes it after it's being used by f function.
// In case f returns no error tx transaction will be committed.
func (m *Measured) InTransaction(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
	return m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		return f(ctx, newMeasuredTx(tx))
	})
}

// Start initializes repository.
func (m *Measured) Start(ctx context.Context) error {
	return m.rep.Start(ctx)
}

// Stop releases all underlying repository resources.
func (m *Measured) Stop(ctx context.Context) error {
	return m.rep.Stop(ctx)
}

func reportOpMetric(opType string, durationInSecs float64, success bool, inTx bool) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"type":     opType,
		"success":  strconv.FormatBool(success),
		"tx":       strconv.FormatBool(inTx),
	}
	repOperations.With(metricLabel).Inc()
	repOperationDurationBucket.With(metricLabel).Observe(durationInSecs)
}
