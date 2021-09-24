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

package rediscache

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
)

// Cache represents a Redis cache type.
type Cache struct {
	rdb *redis.Client
}

// New returns an initialized Cache instance.
func New(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// Get returns the bytes value associated to k.
// If k element is not present, the returned payload will be nil.
func (c *Cache) Get(ctx context.Context, k string) ([]byte, error) {
	b, err := c.rdb.Get(ctx, k).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// Set stores a new element in the memory cache, overwriting it if it was already present.
func (c *Cache) Set(ctx context.Context, k string, b []byte) error {
	return c.rdb.Set(ctx, k, b, 0).Err()
}
