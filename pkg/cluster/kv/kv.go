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

package kv

import "context"

// WatchEventType represents a key watch event type.
type WatchEventType uint8

const (
	// Put represents a put key-value event type.
	Put WatchEventType = iota

	// Del represents a delete key-value event type.
	Del
)

// WatchEvent represents a single watched event.
type WatchEvent struct {
	Type    WatchEventType
	Key     string
	Val     []byte
	PrevVal []byte
}

// WatchResp contains a watch operation response value.
type WatchResp struct {
	Events []WatchEvent
	Err    error
}

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
	Watch(ctx context.Context, prefix string, withPrevVal bool) <-chan WatchResp

	// Start initializes key-value store.
	Start(ctx context.Context) error

	// Stop releases all underlying key-value store resources.
	Stop(ctx context.Context) error
}
