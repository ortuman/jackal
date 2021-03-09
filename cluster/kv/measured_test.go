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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMeasuredKV_Put(t *testing.T) {
	// given
	kvMock := &kvMock{}
	kvMock.PutFunc = func(ctx context.Context, key string, value string) error {
		return nil
	}
	mkv := NewMeasured(kvMock)

	// when
	_ = mkv.Put(context.Background(), "k0", "v0")

	// then
	require.Len(t, kvMock.PutCalls(), 1)
}

func TestMeasuredKV_Get(t *testing.T) {
	// given
	kvMock := &kvMock{}
	kvMock.GetFunc = func(ctx context.Context, key string) ([]byte, error) {
		return nil, nil
	}
	mkv := NewMeasured(kvMock)

	// when
	_, _ = mkv.Get(context.Background(), "k0")

	// then
	require.Len(t, kvMock.GetCalls(), 1)
}

func TestMeasuredKV_GetPrefix(t *testing.T) {
	// given
	kvMock := &kvMock{}
	kvMock.GetPrefixFunc = func(ctx context.Context, prefix string) (map[string][]byte, error) {
		return nil, nil
	}
	mkv := NewMeasured(kvMock)

	// when
	_, _ = mkv.GetPrefix(context.Background(), "i://")

	// then
	require.Len(t, kvMock.GetPrefixCalls(), 1)
}

func TestMeasuredKV_Del(t *testing.T) {
	// given
	kvMock := &kvMock{}
	kvMock.DelFunc = func(ctx context.Context, key string) error {
		return nil
	}
	mkv := NewMeasured(kvMock)

	// when
	_ = mkv.Del(context.Background(), "k0")

	// then
	require.Len(t, kvMock.DelCalls(), 1)
}

func TestMeasuredKV_Watch(t *testing.T) {
	// given
	kvMock := &kvMock{}
	kvMock.WatchFunc = func(ctx context.Context, prefix string, withPrevVal bool) <-chan WatchResp {
		return nil
	}
	mkv := NewMeasured(kvMock)

	// when
	_ = mkv.Watch(context.Background(), "i://", true)

	// then
	require.Len(t, kvMock.WatchCalls(), 1)
}
