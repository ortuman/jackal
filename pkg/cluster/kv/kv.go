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

package kv

import (
	"context"
	"fmt"

	kitlog "github.com/go-kit/log"
	etcdkv "github.com/ortuman/jackal/pkg/cluster/kv/etcd"
	kvtypes "github.com/ortuman/jackal/pkg/cluster/kv/types"
)

const etcdKVType = "etcd"

// KV represents a generic key-value store interface.
type KV interface {
	// Put stores a new value associated to a given key.
	Put(ctx context.Context, key string, value string) error

	// Get retrieves a value associated to a given key.
	Get(ctx context.Context, key string) ([]byte, error)

	// GetPrefix retrieves all values whose key matches prefix.
	GetPrefix(ctx context.Context, prefix string) (map[string][]byte, error)

	// Del deletes a value associated to a given key.
	Del(ctx context.Context, key string) error

	// Watch watches on a key or prefix.
	Watch(ctx context.Context, prefix string, withPrevVal bool) <-chan kvtypes.WatchResp

	// Start initializes key-value store.
	Start(ctx context.Context) error

	// Stop releases all underlying key-value store resources.
	Stop(ctx context.Context) error
}

// Config defines cluster KV configuration.
type Config struct {
	Type string        `fig:"type"`
	Etcd etcdkv.Config `fig:"etcd"`
}

// New returns a new initialized KV instance.
func New(cfg Config, logger kitlog.Logger) (KV, error) {
	switch cfg.Type {
	case etcdKVType:
		return etcdkv.New(cfg.Etcd, logger), nil

	default:
		return nil, fmt.Errorf("unrecognized cluster kv type: %s", cfg.Type)
	}
}
