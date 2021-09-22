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

	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/repository"
)

type measuredRosterRep struct {
	rep repository.Roster
}

func (m *measuredRosterRep) TouchRosterVersion(ctx context.Context, username string) (int, error) {
	t0 := time.Now()
	ver, err := m.rep.TouchRosterVersion(ctx, username)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return ver, err
}

func (m *measuredRosterRep) FetchRosterVersion(ctx context.Context, username string) (int, error) {
	t0 := time.Now()
	ver, err := m.rep.FetchRosterVersion(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return ver, err
}

func (m *measuredRosterRep) UpsertRosterItem(ctx context.Context, ri *rostermodel.Item) error {
	t0 := time.Now()
	err := m.rep.UpsertRosterItem(ctx, ri)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return err
}

func (m *measuredRosterRep) DeleteRosterItem(ctx context.Context, username, jid string) error {
	t0 := time.Now()
	err := m.rep.DeleteRosterItem(ctx, username, jid)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return err
}

func (m *measuredRosterRep) DeleteRosterItems(ctx context.Context, username string) error {
	t0 := time.Now()
	err := m.rep.DeleteRosterItems(ctx, username)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return err
}

func (m *measuredRosterRep) FetchRosterItems(ctx context.Context, username string) ([]*rostermodel.Item, error) {
	t0 := time.Now()
	items, err := m.rep.FetchRosterItems(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return items, err
}

func (m *measuredRosterRep) FetchRosterItemsInGroups(ctx context.Context, username string, groups []string) ([]*rostermodel.Item, error) {
	t0 := time.Now()
	items, err := m.rep.FetchRosterItemsInGroups(ctx, username, groups)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return items, err
}

func (m *measuredRosterRep) FetchRosterItem(ctx context.Context, username, jid string) (*rostermodel.Item, error) {
	t0 := time.Now()
	itm, err := m.rep.FetchRosterItem(ctx, username, jid)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return itm, err
}

func (m *measuredRosterRep) UpsertRosterNotification(ctx context.Context, rn *rostermodel.Notification) error {
	t0 := time.Now()
	err := m.rep.UpsertRosterNotification(ctx, rn)
	reportOpMetric(upsertOp, time.Since(t0).Seconds(), err == nil)
	return err
}

func (m *measuredRosterRep) DeleteRosterNotification(ctx context.Context, contact, jid string) error {
	t0 := time.Now()
	err := m.rep.DeleteRosterNotification(ctx, contact, jid)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return err
}

func (m *measuredRosterRep) DeleteRosterNotifications(ctx context.Context, contact string) error {
	t0 := time.Now()
	err := m.rep.DeleteRosterNotifications(ctx, contact)
	reportOpMetric(deleteOp, time.Since(t0).Seconds(), err == nil)
	return err
}

func (m *measuredRosterRep) FetchRosterNotification(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
	t0 := time.Now()
	rn, err := m.rep.FetchRosterNotification(ctx, contact, jid)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return rn, err
}

func (m *measuredRosterRep) FetchRosterNotifications(ctx context.Context, contact string) ([]*rostermodel.Notification, error) {
	t0 := time.Now()
	rns, err := m.rep.FetchRosterNotifications(ctx, contact)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return rns, err
}

func (m *measuredRosterRep) FetchRosterGroups(ctx context.Context, username string) ([]string, error) {
	t0 := time.Now()
	groups, err := m.rep.FetchRosterGroups(ctx, username)
	reportOpMetric(fetchOp, time.Since(t0).Seconds(), err == nil)
	return groups, err
}
