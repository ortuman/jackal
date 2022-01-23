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

package cachedrepository

import (
	"context"
	"reflect"
	"testing"

	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/stretchr/testify/require"
)

func TestCachedRepository_ExistsOp(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.HasKeyFunc = func(ctx context.Context, k string) (bool, error) {
		if k == "k0" {
			return true, nil
		}
		return false, nil
	}
	missFn := func(context.Context) (bool, error) {
		return false, nil
	}

	// when
	op0 := existsOp{c: cacheMock, key: "k0", missFn: missFn}
	op1 := existsOp{c: cacheMock, key: "k1", missFn: missFn}

	ok0, _ := op0.do(context.Background())
	ok1, _ := op1.do(context.Background())

	// then
	require.True(t, ok0)
	require.False(t, ok1)
}

func TestCachedRepository_UpdateOp(t *testing.T) {
	// given
	var output string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, k string) error {
		output += "del"
		return nil
	}
	updateFn := func(context.Context) error {
		output += ":rep_update"
		return nil
	}

	// when
	op := updateOp{c: cacheMock, key: "k0", updateFn: updateFn}
	_ = op.do(context.Background())

	// then
	require.Equal(t, "del:rep_update", output) // ensure proper order
}

func TestCachedRepository_FetchOpHit(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, k string) ([]byte, error) {
		return []byte{255}, nil
	}

	c := &codecMock{}
	c.decodeFunc = func(bytes []byte) error {
		return nil
	}
	var usr usermodel.User
	c.valueFunc = func() interface{} {
		return &usr
	}

	var missed bool
	op := fetchOp{
		c:     cacheMock,
		codec: c,
		missFn: func(ctx context.Context) (interface{}, error) {
			missed = true
			return nil, nil
		},
	}

	// when
	v, _ := op.do(context.Background())

	// then
	require.False(t, missed)
	require.True(t, reflect.DeepEqual(v, &usr))
}

func TestCachedRepository_FetchOpMiss(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, k string, val []byte) error {
		return nil
	}

	c := &codecMock{}
	c.encodeFunc = func(i interface{}) ([]byte, error) {
		return []byte{255}, nil
	}

	var usr usermodel.User
	op := fetchOp{
		c:     cacheMock,
		codec: c,
		missFn: func(ctx context.Context) (interface{}, error) {
			return &usr, nil
		},
	}

	// when
	v, _ := op.do(context.Background())

	// then
	require.True(t, reflect.DeepEqual(v, &usr))
}
