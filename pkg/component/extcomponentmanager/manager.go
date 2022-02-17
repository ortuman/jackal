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

package extcomponentmanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	clusterconnmanager "github.com/ortuman/jackal/pkg/cluster/connmanager"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	kvtypes "github.com/ortuman/jackal/pkg/cluster/kv/types"
	"github.com/ortuman/jackal/pkg/component"
)

const (
	extComponentKeyPrefix = "ec://"

	extComponentValFormat = "i=%s"

	processEventsTimeout = time.Second * 5
)

// Manager represents external component manager type.
type Manager struct {
	kv             kv.KV
	clusterConnMng clusterConnManager
	comps          components
	ctx            context.Context
	ctxCancel      context.CancelFunc
	logger         kitlog.Logger
}

// New returns a new initialized Manager instance.
func New(kv kv.KV, clusterConnMng *clusterconnmanager.Manager, comps *component.Components, logger kitlog.Logger) *Manager {
	ctx, cancelFn := context.WithCancel(context.Background())
	return &Manager{
		kv:             kv,
		clusterConnMng: clusterConnMng,
		comps:          comps,
		ctx:            ctx,
		ctxCancel:      cancelFn,
		logger:         logger,
	}
}

// RegisterComponentHost registers external component cHost into cluster KV store.
func (m *Manager) RegisterComponentHost(ctx context.Context, cHost string) error {
	return m.kv.Put(ctx, kvComponentHostKey(cHost), fmt.Sprintf(extComponentValFormat, instance.ID()))
}

// UnregisterComponentHost unregisters external component cHost from cluster KV store.
func (m *Manager) UnregisterComponentHost(ctx context.Context, cHost string) error {
	return m.kv.Del(ctx, kvComponentHostKey(cHost))
}

// Start starts external component manager.
func (m *Manager) Start(ctx context.Context) error {
	// fetch external components
	if err := m.refreshExternalComponents(ctx); err != nil {
		return err
	}
	level.Info(m.logger).Log("msg", "started external component manager")
	return nil
}

// Stop stops external component manager.
func (m *Manager) Stop(_ context.Context) error {
	// stop watching changes...
	m.ctxCancel()

	level.Info(m.logger).Log("msg", "stopped external component manager")
	return nil
}

func (m *Manager) refreshExternalComponents(ctx context.Context) error {
	ch := make(chan error, 1)

	go func() {
		wCh := m.kv.Watch(m.ctx, extComponentKeyPrefix, true)

		ecs, err := m.getExtComponents(ctx)
		if err != nil {
			ch <- err
			return
		}
		for _, ec := range ecs {
			if err := m.comps.RegisterComponent(ctx, &ec); err != nil {
				level.Warn(m.logger).Log("msg", "failed to register external component", "err", err)
			}
		}
		close(ch) // signal update

		// watch changes
		for wResp := range wCh {
			if err := wResp.Err; err != nil {
				level.Warn(m.logger).Log("msg", "error occurred watching external components", "err", err)
				continue
			}
			// process change events
			ctx, cancelFn := context.WithTimeout(context.Background(), processEventsTimeout)
			if err := m.processKVEvents(ctx, wResp.Events); err != nil {
				level.Warn(m.logger).Log("msg", "failed to process external component changes", "err", err)
			}
			cancelFn()
		}
	}()
	return <-ch
}

func (m *Manager) getExtComponents(ctx context.Context) ([]extComponent, error) {
	vs, err := m.kv.GetPrefix(ctx, extComponentKeyPrefix)
	if err != nil {
		return nil, err
	}
	res := make([]extComponent, 0, len(vs))
	for k, val := range vs {
		strVal := string(val)
		if isLocalExtComponent(strVal) {
			continue // ignore local external components
		}
		ec, err := m.decodeExtComponent(k, strVal)
		if err != nil {
			level.Warn(m.logger).Log("msg", "failed to decode external component", "err", err)
			continue
		}
		if ec == nil {
			continue // local external component
		}
		res = append(res, *ec)
	}
	return res, nil
}

func (m *Manager) decodeExtComponent(k, val string) (*extComponent, error) {
	cHost := strings.TrimPrefix(k, extComponentKeyPrefix)

	var instanceID string
	_, _ = fmt.Sscanf(val, extComponentValFormat, &instanceID)

	conn, err := m.clusterConnMng.GetConnection(instanceID)
	if err != nil {
		return nil, err
	}
	return newExtComponent(cHost, conn), nil
}

func (m *Manager) processKVEvents(ctx context.Context, kvEvents []kvtypes.WatchEvent) error {
	for _, ev := range kvEvents {
		strVal := string(ev.Val)
		if isLocalExtComponent(strVal) {
			continue // ignore local external components
		}
		switch ev.Type {
		case kvtypes.Put:
			ec, err := m.decodeExtComponent(ev.Key, strVal)
			if err != nil {
				return err
			}
			if err := m.comps.RegisterComponent(ctx, ec); err != nil {
				return err
			}

		case kvtypes.Del:
			cHost := strings.TrimPrefix(ev.Key, extComponentKeyPrefix)
			if err := m.comps.UnregisterComponent(ctx, cHost); err != nil {
				return err
			}
		}
	}
	return nil
}

func kvComponentHostKey(cHost string) string {
	return extComponentKeyPrefix + cHost
}

func isLocalExtComponent(v string) bool {
	return strings.TrimPrefix(v, "i=") == instance.ID()
}
