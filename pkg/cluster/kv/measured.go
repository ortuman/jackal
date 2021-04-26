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

import (
	"context"
	"strconv"
	"time"

	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	putOpType    = "put"
	getOpType    = "get"
	deleteOpType = "delete"
	watchOpType  = "watch"
)

// Measured represents a measured KV type.
type Measured struct {
	kv KV
}

// NewMeasured returns an initialized measured KV instance.
func NewMeasured(kv KV) *Measured {
	return &Measured{kv: kv}
}

// Put stores a new value associated to a given key.
func (m *Measured) Put(ctx context.Context, key string, value string) error {
	t0 := time.Now()
	err := m.kv.Put(ctx, key, value)
	reportMetric(putOpType, time.Since(t0).Seconds(), err == nil)
	return err
}

// Get retrieves a value associated to a given key.
func (m *Measured) Get(ctx context.Context, key string) ([]byte, error) {
	t0 := time.Now()
	v, err := m.kv.Get(ctx, key)
	reportMetric(getOpType, time.Since(t0).Seconds(), err == nil)
	return v, err
}

// GetPrefix retrieves all values whose key matches prefix.
func (m *Measured) GetPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	t0 := time.Now()
	vs, err := m.kv.GetPrefix(ctx, prefix)
	reportMetric(getOpType, time.Since(t0).Seconds(), err == nil)
	return vs, err
}

// Del deletes a value associated to a given key.
func (m *Measured) Del(ctx context.Context, key string) error {
	t0 := time.Now()
	err := m.kv.Del(ctx, key)
	reportMetric(deleteOpType, time.Since(t0).Seconds(), err == nil)
	return err
}

// Watch watches on a key or prefix.
func (m *Measured) Watch(ctx context.Context, prefix string, withPrevVal bool) <-chan WatchResp {
	t0 := time.Now()
	ch := m.kv.Watch(ctx, prefix, withPrevVal)
	reportMetric(watchOpType, time.Since(t0).Seconds(), true)
	return ch
}

// Start initializes key-value store.
func (m *Measured) Start(ctx context.Context) error {
	return m.kv.Start(ctx)
}

// Stop releases all underlying key-value store resources.
func (m *Measured) Stop(ctx context.Context) error {
	return m.kv.Stop(ctx)
}

func reportMetric(opType string, durationInSecs float64, success bool) {
	metricLabel := prometheus.Labels{
		"instance": instance.ID(),
		"type":     opType,
		"success":  strconv.FormatBool(success),
	}
	kvOperations.With(metricLabel).Inc()
	kvOperationDurationBucket.With(metricLabel).Observe(durationInSecs)
}
