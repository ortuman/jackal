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

package rediscache

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/cespare/xxhash"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/go-redis/redis/v8"
	dnsutil "github.com/ortuman/jackal/pkg/util/dns"
)

const resolveInterval = time.Second * 5

type servicePool interface {
	getClient(s string) *redis.Client

	start(ctx context.Context) error
	stop(ctx context.Context) error
}

type staticServicePool struct {
	cfg     Config
	clients []*redis.Client
}

func newStaticServicePool(cfg Config) servicePool {
	return &staticServicePool{cfg: cfg}
}

func (p *staticServicePool) getClient(s string) *redis.Client {
	if len(p.clients) == 1 {
		return p.clients[0]
	}
	cs := xxhash.Sum64String(s)
	idx := jumpHash(cs, len(p.clients))
	return p.clients[idx]
}

func (p *staticServicePool) start(ctx context.Context) error {
	for _, addr := range p.cfg.Addresses {
		client := redis.NewClient(&redis.Options{
			Addr:         addr,
			Username:     p.cfg.Username,
			Password:     p.cfg.Password,
			DB:           p.cfg.DB,
			DialTimeout:  p.cfg.DialTimeout,
			ReadTimeout:  p.cfg.ReadTimeout,
			WriteTimeout: p.cfg.WriteTimeout,
		})
		if err := client.Ping(ctx).Err(); err != nil {
			return err
		}
		p.clients = append(p.clients, client)
	}
	return nil
}

func (p *staticServicePool) stop(_ context.Context) error {
	for _, client := range p.clients {
		_ = client.Close()
	}
	return nil
}

type clientEntry struct {
	addr   string
	client *redis.Client
}

type srvServicePool struct {
	cfg Config

	rsv *dnsutil.SRVResolver

	clientsMu     sync.RWMutex
	clientEntries []clientEntry

	logger log.Logger
}

func newSRVServicePool(cfg Config, logger log.Logger) servicePool {
	return &srvServicePool{
		cfg:    cfg,
		logger: logger,
	}
}

func (p *srvServicePool) getClient(s string) *redis.Client {
	p.clientsMu.RLock()
	defer p.clientsMu.RUnlock()

	if len(p.clientEntries) == 1 {
		return p.clientEntries[0].client
	}
	cs := xxhash.Sum64String(s)
	idx := jumpHash(cs, len(p.clientEntries))
	return p.clientEntries[idx].client
}

func (p *srvServicePool) start(ctx context.Context) error {
	srv, proto, name, err := dnsutil.ParseSRVRecord(p.cfg.SRV)
	if err != nil {
		return err
	}
	p.rsv = dnsutil.NewSRVResolver(srv, proto, name, resolveInterval, p.logger)
	if err := p.rsv.Resolve(ctx); err != nil {
		return err
	}
	if err := p.syncClients(ctx, p.rsv.Targets(), nil); err != nil {
		return err
	}
	go p.runUpdate()
	return nil
}

func (p *srvServicePool) stop(_ context.Context) error {
	p.rsv.Close()

	p.clientsMu.RLock()
	for _, ent := range p.clientEntries {
		_ = ent.client.Close()
	}
	p.clientsMu.RUnlock()
	return nil
}

func (p *srvServicePool) runUpdate() {
	const syncTimeout = time.Second * 5
	for upd := range p.rsv.Update() {
		ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
		if err := p.syncClients(ctx, upd.NewTargets, upd.OldTargets); err != nil {
			level.Warn(p.logger).Log("msg", "failed to synchronize clients", "err", err)
		}
		cancel()
	}
}

func (p *srvServicePool) syncClients(ctx context.Context, newTargets, oldTargets []string) error {
	p.clientsMu.RLock()
	newClientEntries := make([]clientEntry, len(p.clientEntries))
	copy(newClientEntries, p.clientEntries)
	p.clientsMu.RUnlock()

	// disconnect from old targets and remove their entries
	for _, oldTarget := range oldTargets {
		for i, ent := range newClientEntries {
			if ent.addr != oldTarget {
				continue
			}
			_ = ent.client.Close()
			newClientEntries = append(newClientEntries[:i], newClientEntries[i+1:]...)
			break
		}
	}
	// append new targets entries
	for _, newTarget := range newTargets {
		client := redis.NewClient(&redis.Options{
			Addr:         newTarget,
			Username:     p.cfg.Username,
			Password:     p.cfg.Password,
			DB:           p.cfg.DB,
			DialTimeout:  p.cfg.DialTimeout,
			ReadTimeout:  p.cfg.ReadTimeout,
			WriteTimeout: p.cfg.WriteTimeout,
		})
		if err := client.Ping(ctx).Err(); err != nil {
			return err
		}
		p.clientEntries = append(p.clientEntries, clientEntry{
			addr:   newTarget,
			client: client,
		})
	}

	sort.Slice(p.clientEntries, func(i, j int) bool {
		return p.clientEntries[i].addr < p.clientEntries[j].addr
	})
	p.clientsMu.Lock()
	p.clientEntries = newClientEntries
	p.clientsMu.Unlock()

	if len(newTargets) > 0 {
		level.Debug(p.logger).Log("msg", "new SRV targets found", "targets", newTargets)
	}
	if len(oldTargets) > 0 {
		level.Debug(p.logger).Log("msg", "removed old SRV targets", "targets", oldTargets)
	}
	return nil
}
