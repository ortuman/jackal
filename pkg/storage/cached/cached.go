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

package cachedrepository

import (
	"context"
	"fmt"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	rediscache "github.com/ortuman/jackal/pkg/storage/cached/redis"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

// Config contains cached repository configuration.
type Config struct {
	Type  string
	Redis rediscache.Config
}

// Cache defines cache store interface.
type Cache interface {
	// Type identifies underlying cache store type.
	Type() string

	// Get retrieves k value from the cache store.
	// If not present nil will be returned.
	Get(ctx context.Context, k string) ([]byte, error)

	// Put stores a value into the cache store.
	Put(ctx context.Context, k string, val []byte) error

	// Del removes k value from the cache store.
	Del(ctx context.Context, k string) error

	// HasKey tells whether k is present in the cache store.
	HasKey(ctx context.Context, k string) (bool, error)

	// Start starts Cache component.
	Start(ctx context.Context) error

	// Stop stops Cache component.
	Stop(ctx context.Context) error
}

// CachedRepository is cached Repository implementation.
type CachedRepository struct {
	repository.User
	repository.Last
	repository.Capabilities
	repository.Offline
	repository.BlockList
	repository.Private
	repository.Roster
	repository.VCard
	repository.Locker

	rep repository.Repository

	cache  Cache
	logger kitlog.Logger
}

// New returns a new initialized CachedRepository instance.
func New(cfg Config, rep repository.Repository, logger kitlog.Logger) (repository.Repository, error) {
	if cfg.Type != rediscache.Type {
		return nil, fmt.Errorf("unrecognized repository cache type: %s", cfg.Type)
	}
	c := rediscache.New(cfg.Redis)

	return &CachedRepository{
		User:         &cachedUserRep{c: c, rep: rep},
		Last:         rep,
		Capabilities: rep,
		Offline:      rep,
		BlockList:    rep,
		Private:      rep,
		Roster:       rep,
		VCard:        &cachedVCardRep{c: c, rep: rep},
		Locker:       rep,
		rep:          rep,
		cache:        c,
		logger:       logger,
	}, nil
}

// InTransaction generates a repository transaction and completes it after it's being used by f function.
// In case f returns no error tx transaction will be committed.
func (c *CachedRepository) InTransaction(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
	return c.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		return f(ctx, newCacheTx(c.cache, tx))
	})
}

// Start starts cached repository component.
func (c *CachedRepository) Start(ctx context.Context) error {
	if err := c.cache.Start(ctx); err != nil {
		return err
	}
	level.Info(c.logger).Log("msg", "started cached repository", "type", c.cache.Type())
	return c.rep.Start(ctx)
}

// Stop stops cached repository component.
func (c *CachedRepository) Stop(ctx context.Context) error {
	if err := c.cache.Stop(ctx); err != nil {
		return err
	}
	level.Info(c.logger).Log("msg", "stopped cached repository", "type", c.cache.Type())
	return c.rep.Stop(ctx)
}
