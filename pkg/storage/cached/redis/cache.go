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

	"github.com/cespare/xxhash/v2"
	"github.com/go-redis/redis/v8"
	netutil "github.com/ortuman/jackal/pkg/util/net"
)

// Type is redis type identifier.
const Type = "redis"

// Config contains Redis cache configuration.
type Config struct {
	DNS          string        `fig:"dns"`
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
	clients []*redis.Client
	ttl     time.Duration
}

// New creates and returns an initialized Redis Cache instance.
func New(cfg Config) (*Cache, error) {
	rdc := &Cache{ttl: cfg.TTL}

	var addr []string
	if len(cfg.Addresses) > 0 {
		addr = cfg.Addresses
	} else {
		var err error
		addr, err = netutil.SRVResolve("tcp", "tcp", cfg.DNS)
		if err != nil {
			return nil, err
		}
	}
	for _, addr := range addr {
		rdc.clients = append(rdc.clients, redis.NewClient(&redis.Options{
			Addr:         addr,
			Username:     cfg.Username,
			Password:     cfg.Password,
			DB:           cfg.DB,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		}))
	}
	return rdc, nil
}

// Type satisfies Cache interface.
func (c *Cache) Type() string { return Type }

// Get satisfies Cache interface.
func (c *Cache) Get(ctx context.Context, ns, key string) ([]byte, error) {
	cl := c.pickClient(ns)
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
	cl := c.pickClient(ns)
	if err := cl.HSet(ctx, ns, key, val).Err(); err != nil {
		return err
	}
	return cl.Expire(ctx, ns, c.ttl).Err()
}

// Del satisfies Cache interface.
func (c *Cache) Del(ctx context.Context, ns string, keys ...string) error {
	cl := c.pickClient(ns)
	return cl.HDel(ctx, ns, keys...).Err()
}

// DelNS removes all keys contained under a given namespace from the cache store.
func (c *Cache) DelNS(ctx context.Context, ns string) error {
	cl := c.pickClient(ns)
	return cl.Del(ctx, ns).Err()
}

// HasKey satisfies Cache interface.
func (c *Cache) HasKey(ctx context.Context, ns, key string) (bool, error) {
	cl := c.pickClient(ns)
	res := cl.HExists(ctx, ns, key)
	if err := res.Err(); err != nil {
		return false, err
	}
	return res.Val(), nil
}

// Start satisfies Cache interface.
func (c *Cache) Start(ctx context.Context) error {
	for _, cl := range c.clients {
		if err := cl.Ping(ctx).Err(); err != nil {
			return err
		}
	}
	return nil
}

// Stop satisfies Cache interface.
func (c *Cache) Stop(_ context.Context) error {
	for _, cl := range c.clients {
		if err := cl.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cache) pickClient(ns string) *redis.Client {
	if len(c.clients) == 1 {
		return c.clients[0]
	}
	cs := xxhash.Sum64String(ns)
	idx := jumpHash(cs, len(c.clients))
	return c.clients[idx]
}
