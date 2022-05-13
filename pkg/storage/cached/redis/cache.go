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
	"errors"
	"time"

	"github.com/go-kit/log"

	"github.com/go-redis/redis/v8"
)

// Type is redis type identifier.
const Type = "redis"

// Config contains Redis cache configuration.
type Config struct {
	SRV          string        `fig:"srv"`
	Addresses    []string      `fig:"addresses"`
	Username     string        `fig:"username"`
	Password     string        `fig:"password"`
	DB           int           `fig:"db"`
	DialTimeout  time.Duration `fig:"dial_timeout" default:"3s"`
	ReadTimeout  time.Duration `fig:"read_timeout" default:"5s"`
	WriteTimeout time.Duration `fig:"write_timeout" default:"5s"`
	TTL          time.Duration `fig:"ttl" default:"24h"`
}

// Cache is Redis cache implementation.
type Cache struct {
	cfg    Config
	sp     servicePool
	ttl    time.Duration
	logger log.Logger
}

// New creates and returns an initialized Redis Cache instance.
func New(cfg Config, logger log.Logger) *Cache {
	return &Cache{
		cfg:    cfg,
		logger: log.With(logger, "cache", "redis"),
	}
}

// Type satisfies Cache interface.
func (c *Cache) Type() string { return Type }

// Get satisfies Cache interface.
func (c *Cache) Get(ctx context.Context, ns, key string) ([]byte, error) {
	cl := c.sp.getClient(ns)
	val, err := cl.HGet(ctx, ns, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return []byte(val), nil
}

// Put satisfies Cache interface.
func (c *Cache) Put(ctx context.Context, ns, key string, val []byte) error {
	cl := c.sp.getClient(ns)
	if err := cl.HSet(ctx, ns, key, val).Err(); err != nil {
		return err
	}
	return cl.Expire(ctx, ns, c.ttl).Err()
}

// Del satisfies Cache interface.
func (c *Cache) Del(ctx context.Context, ns string, keys ...string) error {
	cl := c.sp.getClient(ns)
	return cl.HDel(ctx, ns, keys...).Err()
}

// DelNS removes all keys contained under a given namespace from the cache store.
func (c *Cache) DelNS(ctx context.Context, ns string) error {
	cl := c.sp.getClient(ns)
	return cl.Del(ctx, ns).Err()
}

// HasKey satisfies Cache interface.
func (c *Cache) HasKey(ctx context.Context, ns, key string) (bool, error) {
	cl := c.sp.getClient(ns)
	res := cl.HExists(ctx, ns, key)
	if err := res.Err(); err != nil {
		return false, err
	}
	return res.Val(), nil
}

// Start satisfies Cache interface.
func (c *Cache) Start(ctx context.Context) error {
	if len(c.cfg.SRV) > 0 {
		c.sp = newSRVServicePool(c.cfg, c.logger)
	} else {
		c.sp = newStaticServicePool(c.cfg)
	}
	return c.sp.start(ctx)
}

// Stop satisfies Cache interface.
func (c *Cache) Stop(ctx context.Context) error {
	return c.sp.stop(ctx)
}
