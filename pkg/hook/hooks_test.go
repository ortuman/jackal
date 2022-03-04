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

package hook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHooks_Add(t *testing.T) {
	// given
	h := NewHooks()

	// when
	h.AddHook("h1", nil, 10)
	h.AddHook("h1", nil, 1)
	h.AddHook("h1", nil, 3)

	// then
	require.Len(t, h.handlers["h1"], 3)

	require.Equal(t, Priority(10), h.handlers["h1"][0].p)
	require.Equal(t, Priority(3), h.handlers["h1"][1].p)
	require.Equal(t, Priority(1), h.handlers["h1"][2].p)
}

func TestHooks_Remove(t *testing.T) {
	// given
	h := NewHooks()

	// when
	var hnd1 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { return nil }
	var hnd2 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { return nil }
	var hnd3 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { return nil }

	h.AddHook("h1", hnd1, 0)
	h.AddHook("h1", hnd2, 0)
	h.AddHook("h1", hnd3, 0)

	h.RemoveHook("h1", hnd3)
	h.RemoveHook("h1", hnd2)
	h.RemoveHook("h1", hnd1)

	// then
	require.Len(t, h.handlers["h1"], 0)
}

func TestHooks_Run(t *testing.T) {
	// given
	h := NewHooks()

	// when
	var i int
	var hnd1 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { i++; return nil }
	var hnd2 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { i++; return nil }
	var hnd3 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { i++; return nil }

	h.AddHook("h1", hnd1, 0)
	h.AddHook("h1", hnd2, 0)
	h.AddHook("h1", hnd3, 0)

	halted, err := h.Run(context.Background(), "h1", nil)

	// then
	require.Nil(t, err)
	require.False(t, halted)

	require.Equal(t, 3, i)
}

func TestHooks_HaltedRun(t *testing.T) {
	// given
	h := NewHooks()

	// when
	var i int
	var hnd1 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { i++; return nil }
	var hnd2 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { i++; return ErrStopped }
	var hnd3 Handler = func(ctx context.Context, execCtx *ExecutionContext) error { i++; return nil }

	h.AddHook("h1", hnd1, 10)
	h.AddHook("h1", hnd2, 5)
	h.AddHook("h1", hnd3, 0)

	halted, err := h.Run(context.Background(), "h1", nil)

	// then
	require.Nil(t, err)
	require.True(t, halted)

	require.Equal(t, 2, i)
}
