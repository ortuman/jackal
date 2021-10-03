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

package repositorycache

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
)

const redisCacheTypeName = "redis"

// RedisCache represents a Redis cache type.
type RedisCache struct {
	rdb *redis.Client
}

// NewRedisCache returns an initialized Cache instance.
func NewRedisCache(rdb *redis.Client) *RedisCache {
	return &RedisCache{rdb: rdb}
}

// Fetch returns the bytes value associated to k.
// If k element is not present the returned payload will be nil.
func (c *RedisCache) Fetch(ctx context.Context, k string) ([]byte, error) {
	b, err := c.rdb.Get(ctx, k).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// Store stores a new element in the memory cache, overwriting it if already present.
func (c *RedisCache) Store(ctx context.Context, k string, b []byte) error {
	return c.rdb.Set(ctx, k, b, 0).Err()
}

// Del removes k associated element from the memory cache.
func (c *RedisCache) Del(ctx context.Context, k string) error {
	return c.rdb.Del(ctx, k).Err()
}

// Exists returns true in case k element is present in the cache.
func (c *RedisCache) Exists(ctx context.Context, k string) (bool, error) {
	val, err := c.rdb.Exists(ctx, k).Result()
	if err != nil {
		return false, err
	}
	return val == 1, nil
}

// Type returns a human-readable cache type name.
func (c *RedisCache) Type() string {
	return redisCacheTypeName
}
