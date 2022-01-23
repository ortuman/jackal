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

package kv

import "context"

// NewNopKV returns a KV that doesn't do anything.
func NewNopKV() KV { return &nopKV{} }

// IsNop tells whether kvs is nop KV.
func IsNop(kv KV) bool {
	_, ok := kv.(*nopKV)
	return ok
}

type nopKV struct{}

func (k *nopKV) Put(_ context.Context, _ string, _ string) error { return nil }
func (k *nopKV) Get(_ context.Context, _ string) ([]byte, error) { return nil, nil }

func (k *nopKV) GetPrefix(_ context.Context, _ string) (map[string][]byte, error) {
	return nil, nil
}

func (k *nopKV) Del(_ context.Context, _ string) error { return nil }

func (k *nopKV) Watch(_ context.Context, _ string, _ bool) <-chan WatchResp {
	// return an already closed channel
	retCh := make(chan WatchResp)
	close(retCh)
	return retCh
}

func (k *nopKV) Start(_ context.Context) error { return nil }
func (k *nopKV) Stop(_ context.Context) error  { return nil }
